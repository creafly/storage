package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/creafly/storage/internal/domain/entity"
	"github.com/stretchr/testify/mock"
)

type FileRepositoryInterface interface {
	Create(ctx context.Context, file *entity.File) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID, fileType *entity.FileType, limit, offset int) ([]entity.File, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateURL(ctx context.Context, id uuid.UUID, url string) error
}

type MockFileRepository struct {
	mock.Mock
}

func (m *MockFileRepository) Create(ctx context.Context, file *entity.File) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockFileRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.File), args.Error(1)
}

func (m *MockFileRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, fileType *entity.FileType, limit, offset int) ([]entity.File, error) {
	args := m.Called(ctx, tenantID, fileType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.File), args.Error(1)
}

func (m *MockFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileRepository) UpdateURL(ctx context.Context, id uuid.UUID, url string) error {
	args := m.Called(ctx, id, url)
	return args.Error(0)
}

type MinioClientInterface interface {
	Upload(ctx context.Context, tenantID uuid.UUID, fileName string, contentType string, data []byte) (string, error)
	Download(ctx context.Context, objectPath string) ([]byte, error)
	Delete(ctx context.Context, objectPath string) error
	GetPresignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error)
	GetPublicURL(objectPath string) string
}

type MockMinioClient struct {
	mock.Mock
}

func (m *MockMinioClient) Upload(ctx context.Context, tenantID uuid.UUID, fileName string, contentType string, data []byte) (string, error) {
	args := m.Called(ctx, tenantID, fileName, contentType, data)
	return args.String(0), args.Error(1)
}

func (m *MockMinioClient) Download(ctx context.Context, objectPath string) ([]byte, error) {
	args := m.Called(ctx, objectPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockMinioClient) Delete(ctx context.Context, objectPath string) error {
	args := m.Called(ctx, objectPath)
	return args.Error(0)
}

func (m *MockMinioClient) GetPresignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, objectPath, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockMinioClient) GetPublicURL(objectPath string) string {
	args := m.Called(objectPath)
	return args.String(0)
}
