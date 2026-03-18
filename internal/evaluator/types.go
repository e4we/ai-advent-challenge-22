package evaluator

// Question — контрольный вопрос для оценки RAG-пайплайна.
type Question struct {
	Text            string   `json:"text"`
	ExpectedFacts   []string `json:"expected_facts"`
	ExpectedSources []string `json:"expected_sources"`
}

// QuestionResult — результат оценки одного вопроса.
type QuestionResult struct {
	Question             string       `json:"question"`
	ExpectedFacts        []string     `json:"expected_facts"`
	RAGAnswer            string       `json:"rag_answer"`
	BaselineAnswer       string       `json:"baseline_answer"`
	RAGFactHits          int          `json:"rag_fact_hits"`
	BaselineFactHits     int          `json:"baseline_fact_hits"`
	RAGMatchedFacts      []string     `json:"rag_matched_facts"`
	BaselineMatchedFacts []string     `json:"baseline_matched_facts"`
	Sources              []SourceInfo `json:"sources"`
	Winner               string       `json:"winner"`
	RAGError             string       `json:"rag_error,omitempty"`
	BaselineError        string       `json:"baseline_error,omitempty"`

	// Reranked path (заполняется только если реранкер включён)
	RAGRerankedAnswer       string     `json:"rag_reranked_answer,omitempty"`
	RAGRerankedFactHits     int        `json:"rag_reranked_fact_hits,omitempty"`
	RAGRerankedMatchedFacts []string   `json:"rag_reranked_matched_facts,omitempty"`
	RAGRerankedSources      []SourceInfo `json:"rag_reranked_sources,omitempty"`
	RAGRerankedError        string     `json:"rag_reranked_error,omitempty"`
}

// SourceInfo — информация об источнике из поиска.
type SourceInfo struct {
	File    string  `json:"file"`
	Section string  `json:"section"`
	Score   float32 `json:"score"`
}

// EvalReport — итоговый отчёт оценки.
type EvalReport struct {
	Timestamp           string           `json:"timestamp"`
	Model               string           `json:"model"`
	EmbeddingModel      string           `json:"embedding_model"`
	Collection          string           `json:"collection"`
	TopK                int              `json:"top_k"`
	TotalQuestions      int              `json:"total_questions"`
	RAGWins             int              `json:"rag_wins"`
	BaselineWins        int              `json:"baseline_wins"`
	RAGRerankedWins     int              `json:"rag_reranked_wins,omitempty"`
	Ties                int              `json:"ties"`
	RAGAvgFactsPct      float64          `json:"rag_avg_facts_pct"`
	BaselineAvgFactsPct float64          `json:"baseline_avg_facts_pct"`
	RAGRerankedAvgFactsPct float64       `json:"rag_reranked_avg_facts_pct,omitempty"`
	RerankerEnabled     bool             `json:"reranker_enabled"`
	Duration            string           `json:"duration"`
	Results             []QuestionResult `json:"results"`
}

// Config — конфигурация evaluator.
type Config struct {
	TopK            int
	FetchTopK       int // top-K для реранкера (default 20)
	RerankerEnabled bool
	OutputPath      string
	Model           string
	EmbeddingModel  string
	Collection      string
}
