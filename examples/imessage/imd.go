package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	_ = godotenv.Load()
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func doIndexRequest(ctx context.Context, body map[string]any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		os.Getenv("OPERAND_ENDPOINT")+"/v3/objects",
		bytes.NewReader(buf),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", os.Getenv("OPERAND_API_KEY"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		buf, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d (%s)", resp.StatusCode, string(buf))
	}

	return nil
}

func storeFile(ctx context.Context, client *s3.S3, path string, buf io.ReadSeeker) (string, error) {
	obj := s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(path),
		Body:   buf,
		ACL:    aws.String("public-read"),
	}
	_, err := client.PutObjectWithContext(ctx, &obj)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET"), path), nil
}

func handleIncoming(storage *s3.S3) http.HandlerFunc {
	type request struct {
		From           string  `json:"from"`
		Message        string  `json:"message"`
		Attachment     []byte  `json:"attachment,omitempty"`
		AttachmentType string  `json:"attachment_type,omitempty"`
		Token          *string `json:"token,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		req.Message = strings.Map(func(r rune) rune {
			if unicode.IsPrint(r) {
				return r
			}
			return -1
		}, req.Message)

		if req.Message != "" {
			body := map[string]any{
				"type": "text",
				"metadata": map[string]any{
					"text": req.Message,
				},
				"properties": map[string]any{
					"from": req.From,
				},
				"label": strconv.FormatInt(time.Now().Unix(), 10),
			}
			if v, ok := os.LookupEnv("OPERAND_PARENT_ID"); ok {
				body["parentId"] = v
			}
			if err := doIndexRequest(r.Context(), body); err != nil {
				fmt.Printf("error: %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if storage != nil && req.Attachment != nil && req.AttachmentType != "" {
			var body map[string]any
			switch req.AttachmentType {
			case "image/jpeg", "image/png":
				url, err := storeFile(
					r.Context(),
					storage,
					fmt.Sprintf("/imessage/image/%s", uuid.NewString()),
					bytes.NewReader(req.Attachment),
				)
				if err != nil {
					fmt.Printf("error: %v\n", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				body = map[string]any{
					"type": "image",
					"metadata": map[string]any{
						"imageUrl": url,
					},
					"properties": map[string]any{
						"from": req.From,
					},
					"label": strconv.FormatInt(time.Now().Unix(), 10),
				}
				if v, ok := os.LookupEnv("OPERAND_PARENT_ID"); ok {
					body["parentId"] = v
				}
			case "application/pdf":
				url, err := storeFile(
					r.Context(),
					storage,
					fmt.Sprintf("/imessage/pdf/%s", uuid.NewString()),
					bytes.NewReader(req.Attachment),
				)
				if err != nil {
					fmt.Printf("error: %v\n", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				body = map[string]any{
					"type": "pdf",
					"metadata": map[string]any{
						"pdfUrl": url,
					},
					"properties": map[string]any{
						"from": req.From,
					},
					"label": strconv.FormatInt(time.Now().Unix(), 10),
				}
				if v, ok := os.LookupEnv("OPERAND_PARENT_ID"); ok {
					body["parentId"] = v
				}
			default:
				fmt.Printf("got unsupported attachment type %s\n", req.AttachmentType)
			}

			if body != nil {
				if err := doIndexRequest(r.Context(), body); err != nil {
					fmt.Printf("error: %v\n", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}
}

func run() error {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	// If we have credentials for S3, use them to create a client.
	// This allows the server to support indexing of attachments.
	var (
		s3Client   *s3.S3
		s3Key      = os.Getenv("S3_KEY")
		s3Secret   = os.Getenv("S3_SECRET")
		s3Endpoint = os.Getenv("S3_ENDPOINT")
		s3Region   = os.Getenv("S3_REGION")
		s3Bucket   = os.Getenv("S3_BUCKET")
	)
	if s3Key != "" && s3Secret != "" && s3Endpoint != "" && s3Region != "" && s3Bucket != "" {
		sess := session.New(&aws.Config{
			Credentials: credentials.NewStaticCredentials(s3Key, s3Secret, ""),
			Endpoint:    aws.String(s3Endpoint),
			Region:      aws.String(s3Region),
		})
		s3Client = s3.New(sess)
	}

	http.HandleFunc("/", handleIncoming(s3Client))
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil)
}
