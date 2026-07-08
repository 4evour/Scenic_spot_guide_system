package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

func (s *RAGService) LoadKnowledgeFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件 %s 失败: %v", filePath, err)
	}

	lines := strings.Split(string(data), "\n")
	loadedCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var chunk ChunkData
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			slog.Warn("跳过无法解析的知识行", "error", err, "line_preview", line[:min(len(line), 80)])
			continue
		}
		normalizeKnowledgeChunk(&chunk)
		if chunk.Content == "" {
			continue
		}

		if _, err := s.upsertChunkData(&chunk); err != nil {
			return fmt.Errorf("写入知识片段失败: %v", err)
		}

		loadedCount++
	}

	if loadedCount > 0 {
		s.invalidateKnowledgeCaches()
	}
	return nil
}

func (s *RAGService) LoadKnowledgeFromJSONL(data []byte) (int, error) {
	return s.LoadKnowledgeJSON(data)
}

func (s *RAGService) upsertChunkData(chunk *ChunkData) (*model.KnowledgeChunk, error) {
	normalizeKnowledgeChunk(chunk)
	if chunk.Content == "" {
		return nil, fmt.Errorf("knowledge content cannot be empty")
	}

	vector, err := s.GenerateEmbedding(chunk.Content)
	if err != nil {
		vector = s.bm25FallbackVector(chunk.Content)
	}

	metadataJSON, _ := json.Marshal(chunk.Metadata)
	knowledge := &model.KnowledgeChunk{
		ID:                chunk.ID,
		Content:           chunk.Content,
		Source:            chunk.Source,
		Title:             chunk.Title,
		Metadata:          string(metadataJSON),
		KnowledgeCategory: chunk.KnowledgeCategory,
		SpotID:            chunk.SpotID,
		SpotCategory:      chunk.SpotCategory,
		Vector:            vector,
	}

	exists, err := s.repo.Exists(chunk.ID)
	if err != nil {
		return nil, fmt.Errorf("check knowledge id failed: %v", err)
	}
	if exists {
		if err := s.repo.Update(knowledge); err != nil {
			return nil, fmt.Errorf("update knowledge failed: %v", err)
		}
		return knowledge, nil
	}

	if err := s.repo.Create(knowledge); err != nil {
		return nil, fmt.Errorf("create knowledge failed: %v", err)
	}
	return knowledge, nil
}

func (s *RAGService) CreateKnowledge(input KnowledgeUpsertInput) (*model.KnowledgeChunk, error) {
	chunk := ChunkData{
		ID:                input.ID,
		Title:             input.Title,
		Source:            input.Source,
		Content:           input.Content,
		KnowledgeCategory: input.KnowledgeCategory,
		SpotID:            input.SpotID,
		SpotCategory:      input.SpotCategory,
		Metadata:          input.Metadata,
	}
	knowledge, err := s.upsertChunkData(&chunk)
	if err != nil {
		return nil, err
	}
	s.invalidateKnowledgeCaches()
	return knowledge, nil
}

func (s *RAGService) UpdateKnowledge(id string, input KnowledgeUpsertInput) (*model.KnowledgeChunk, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("knowledge id cannot be empty")
	}
	input.ID = id
	return s.CreateKnowledge(input)
}

func (s *RAGService) LoadKnowledgeDocument(filename string, data []byte, category string) (int, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jsonl":
		return s.LoadKnowledgeJSONLines(data)
	case ".json":
		return s.LoadKnowledgeJSON(data)
	case ".md", ".markdown", ".txt":
		return s.LoadPlainTextKnowledge(filename, string(data), category)
	default:
		return 0, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func (s *RAGService) LoadKnowledgeJSON(data []byte) (int, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var chunks []ChunkData
		if err := json.Unmarshal([]byte(trimmed), &chunks); err != nil {
			return 0, fmt.Errorf("parse JSON failed: %v", err)
		}
		loadedCount := 0
		for _, chunk := range chunks {
			if _, err := s.upsertChunkData(&chunk); err != nil {
				return loadedCount, err
			}
			loadedCount++
		}
		if loadedCount > 0 {
			s.invalidateKnowledgeCaches()
		}
		return loadedCount, nil
	}
	return s.LoadKnowledgeJSONLines(data)
}

func (s *RAGService) LoadKnowledgeJSONLines(data []byte) (int, error) {
	lines := strings.Split(string(data), "\n")
	loadedCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var chunk ChunkData
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return loadedCount, fmt.Errorf("parse JSON line failed: %v", err)
		}
		if _, err := s.upsertChunkData(&chunk); err != nil {
			return loadedCount, err
		}
		loadedCount++
	}
	if loadedCount > 0 {
		s.invalidateKnowledgeCaches()
	}
	return loadedCount, nil
}

func (s *RAGService) LoadPlainTextKnowledge(filename, content, category string) (int, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0, fmt.Errorf("knowledge content cannot be empty")
	}

	title := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	paragraphs := splitKnowledgeParagraphs(content, 1200)
	loadedCount := 0
	for i, paragraph := range paragraphs {
		metadata := map[string]interface{}{
			"filename": filename,
			"chunk":    i + 1,
		}
		if strings.TrimSpace(category) != "" {
			metadata["category"] = strings.TrimSpace(category)
		}

		chunk := ChunkData{
			Title:             fmt.Sprintf("%s-%02d", title, i+1),
			Source:            filename,
			Content:           paragraph,
			KnowledgeCategory: strings.TrimSpace(category),
			Metadata:          metadata,
		}
		if _, err := s.upsertChunkData(&chunk); err != nil {
			return loadedCount, err
		}
		loadedCount++
	}
	if loadedCount > 0 {
		s.invalidateKnowledgeCaches()
	}
	return loadedCount, nil
}

func splitKnowledgeParagraphs(content string, maxRunes int) []string {
	blocks := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n\n")
	chunks := make([]string, 0)
	var current strings.Builder

	flush := func() {
		text := strings.TrimSpace(current.String())
		if text != "" {
			chunks = append(chunks, text)
		}
		current.Reset()
	}

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if current.Len() > 0 && len([]rune(current.String()+"\n\n"+block)) > maxRunes {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(block)
	}
	flush()

	if len(chunks) == 0 {
		return []string{content}
	}
	return chunks
}

func (s *RAGService) SaveUploadedFile(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jsonl", ".json", ".md", ".markdown", ".txt":
	default:
		return "", fmt.Errorf("only .jsonl, .json, .md, .markdown and .txt files are supported")
	}
	if err := os.MkdirAll(s.uploadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	// 使用时间戳前缀避免文件名冲突
	saveName := fmt.Sprintf("%d_%s", time.Now().UnixMilli(), filepath.Base(filename))
	savePath := filepath.Join(s.uploadDir, saveName)
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return savePath, nil
}

func (s *RAGService) DeleteKnowledge(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.invalidateKnowledgeCaches()
	return nil
}

func (s *RAGService) DeleteAllKnowledge() error {
	if err := s.repo.DeleteAll(); err != nil {
		return err
	}
	s.invalidateKnowledgeCaches()
	return nil
}

func (s *RAGService) ListKnowledge(page, pageSize int, keyword, category string) ([]model.KnowledgeChunk, int64, error) {
	return s.repo.List(page, pageSize, keyword, category)
}

func (s *RAGService) ListKnowledgeAdvanced(filter repository.KnowledgeListFilter) ([]model.KnowledgeChunk, int64, error) {
	return s.repo.ListAdvanced(filter)
}

func (s *RAGService) GetKnowledge(id string) (*model.KnowledgeChunk, error) {
	return s.repo.GetByID(id)
}
