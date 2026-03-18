package evaluator

// DefaultQuestions возвращает 10 контрольных вопросов для полной оценки RAG-пайплайна.
func DefaultQuestions() []Question {
	return []Question{
		{
			Text:            "Что такое декораторы в Python?",
			ExpectedFacts:   []string{"функция", "принимает другую функцию", "расширяет", "логирование", "кэширование"},
			ExpectedSources: []string{"python_basics.md"},
		},
		{
			Text:            "Как работают генераторы в Python?",
			ExpectedFacts:   []string{"yield", "ленивые вычисления", "экономия памяти", "бесконечные последовательности"},
			ExpectedSources: []string{"python_basics.md"},
		},
		{
			Text:            "Что такое контекстные менеджеры?",
			ExpectedFacts:   []string{"with", "__enter__", "__exit__", "ресурсы"},
			ExpectedSources: []string{"python_basics.md"},
		},
		{
			Text:            "Какие ключевые компоненты обучения с подкреплением?",
			ExpectedFacts:   []string{"агент", "среда", "состояние", "действие", "награда"},
			ExpectedSources: []string{"machine_learning.md"},
		},
		{
			Text:            "Какие метрики используются для оценки классификации?",
			ExpectedFacts:   []string{"accuracy", "precision", "recall", "F1-score", "ROC-AUC"},
			ExpectedSources: []string{"machine_learning.md"},
		},
		{
			Text:            "Из каких компонентов состоит трансформер?",
			ExpectedFacts:   []string{"encoder", "decoder", "multi-head attention", "positional encoding"},
			ExpectedSources: []string{"machine_learning.md"},
		},
		{
			Text:            "Что утверждает теорема CAP?",
			ExpectedFacts:   []string{"consistency", "availability", "partition tolerance"},
			ExpectedSources: []string{"distributed_systems.md"},
		},
		{
			Text:            "Какие стратегии кэширования существуют?",
			ExpectedFacts:   []string{"cache-aside", "write-through", "write-back", "read-through"},
			ExpectedSources: []string{"distributed_systems.md"},
		},
		{
			Text:            "Как работает Circuit Breaker?",
			ExpectedFacts:   []string{"closed", "open", "half-open", "каскадные отказы"},
			ExpectedSources: []string{"distributed_systems.md"},
		},
		{
			Text:            "В чём разница между fixed-size и structural chunking?",
			ExpectedFacts:   []string{"предсказуемый размер", "разрывать смысловые единицы", "структуру документа", "семантические границы"},
			ExpectedSources: []string{"distributed_systems.md"},
		},
	}
}

// QuickQuestions возвращает 3 вопроса из разных документов для быстрого тестирования.
// Покрывает все 3 документа при минимальном числе API-вызовов (6 вместо 20).
func QuickQuestions() []Question {
	all := DefaultQuestions()
	return []Question{all[0], all[3], all[6]}
}
