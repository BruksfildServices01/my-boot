package handlers

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/gif"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type UploadHandler struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewUploadHandler() *UploadHandler {
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET_NAME")
	publicURL := strings.TrimRight(os.Getenv("R2_PUBLIC_URL"), "/")

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, _ := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
		config.WithRegion("auto"),
	)

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &UploadHandler{client: client, bucket: bucket, publicURL: publicURL}
}

func (u *UploadHandler) Upload(file multipart.File, header *multipart.FileHeader) (string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("leitura do arquivo: %w", err)
	}

	processed, err := processImage(data)
	if err != nil {
		// não é imagem ou formato não suportado — sobe o original sem alteração
		processed = data
	}

	key := fmt.Sprintf("products/%d.jpg", time.Now().UnixNano())

	_, err = u.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(processed),
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		return "", fmt.Errorf("upload R2: %w", err)
	}

	return u.publicURL + "/" + key, nil
}

// processImage redimensiona a imagem para no máximo 1200px de largura
// e recodifica como JPEG com qualidade 85. Aceita JPEG, PNG, GIF e WebP.
func processImage(data []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	const maxWidth = 1200
	bounds := src.Bounds()
	origW, origH := bounds.Dx(), bounds.Dy()

	dstW, dstH := origW, origH
	if origW > maxWidth {
		dstW = maxWidth
		dstH = origH * maxWidth / origW
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
