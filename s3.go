package main

import (
	"bytes"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nfnt/resize"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// NewAWSSession builds a session using the SDK's default credential chain,
// so on ECS it picks up the task role automatically (and falls back to
// AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY or ~/.aws for local development).
func NewAWSSession(region string) (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
}

func UploadFileToS3(s *session.Session, file []byte, name string) (string, error) {
	bucket := config("S3_BUCKET")
	region := *s.Config.Region

	timestamp := strconv.Itoa(int(time.Now().Unix()))
	fileName := filepath.Base(name) + "." + timestamp + filepath.Ext(name)
	maxAgeOneYear := aws.String("max-age=31536000")

	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(bucket),
		Key:                aws.String(fileName),
		ACL:                aws.String("public-read"),
		Body:               bytes.NewReader(file),
		ContentType:        aws.String(http.DetectContentType(file)),
		ContentDisposition: aws.String("attachment"),
		StorageClass:       aws.String("STANDARD"),
		Metadata:           map[string]*string{"Cache-Control": maxAgeOneYear}, // 1 year
	})
	if err != nil {
		return "", err
	}

	return "https://" + bucket + ".s3." + region + ".amazonaws.com/" + fileName, nil
}

// ResizeJPG takes a multipart.File that is expected to be a jpg
// and the width to resize it to in pixels.
// It returns a []byte containing a resized version of the file
// or an error.
func ResizeJPG(file multipart.File, maxWidth uint) ([]byte, error) {
	srcBuf := new(bytes.Buffer)
	if _, err := io.Copy(srcBuf, file); err != nil {
		return nil, err
	}

	srcImg, err := jpeg.Decode(srcBuf)
	if err != nil {
		return nil, err
	}

	dstImg := resize.Resize(maxWidth, 0, srcImg, resize.Lanczos3)

	dstBuf := new(bytes.Buffer)
	if err := jpeg.Encode(dstBuf, dstImg, nil); err != nil {
		return nil, err
	}

	return dstBuf.Bytes(), nil
}
