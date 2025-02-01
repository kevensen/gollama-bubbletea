# Gollama Bubbletea

<div align="center">
  <img src="images/Icon.png" alt="Icon" width="200"/>
</div>

This project provides a text user interface for chatting with Ollama LLM's using the [Bubble Tea Library](https://github.com/charmbracelet/bubbletea) and [Parakeet](https://github.com/parakeet-nest/parakeet/llm) for interfacing with Ollama.

## Execution
```
go run cmd/main.go -ollama_host=http://127.0.0.1 -ollama_port=11434
```

## Things I want to do
- [ ] Add unit tests
- [ ] Add agent support
- [ ] Have the response from the model be written out as it is responding and not 
      all at once