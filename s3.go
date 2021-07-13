package main

import (
	"bytes"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func NewAWSSession(region, auth, secret string) (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			config("S3_AUTH_ID"),
			config("S3_SECRET"),
			"",
		),
	})
}

func UploadFileToS3(s *session.Session, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	bucket := config("S3_BUCKET")
	region := config("S3_REGION")

	size := fileHeader.Size
	buffer := make([]byte, size)
	if _, err := file.Read(buffer); err != nil {
		return "", err
	}

	fileName, err := nonce()
	if err != nil {
		return "", err
	}
	fileName += filepath.Ext(fileHeader.Filename)

	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(bucket),
		Key:                aws.String(fileName),
		ACL:                aws.String("public-read"),
		Body:               bytes.NewReader(buffer),
		ContentLength:      aws.Int64(size),
		ContentType:        aws.String(http.DetectContentType(buffer)),
		ContentDisposition: aws.String("attachment"),
		//ServerSideEncryption: aws.String("AES256"),
		StorageClass: aws.String("STANDARD"),
	})
	if err != nil {
		return "", err
	}

	log.Printf("File uploaded to S3: %v\n", fileName)

	return "https://" + bucket + ".s3." + region + ".amazonaws.com/" + fileName, nil
}
