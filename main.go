package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
)

// Types etc

type ModelResponse struct {
	Model        string
	TotalTokens  int
	Citations    []string
	Content      string
	FinishReason string
}

// Globals

const perplexityApiKey = "PERPLEXITY_API_KEY"
const GeminiApiKey = "GEMINI_API_KEY"
const chatGPTApiKey = "OPENAI_API_KEY"

// TODO: 'finished due to: ' in Gemini output doesn't work

// CheckInternetHTTP attempts to make an HTTP GET request to a reliable server.
// It uses a timeout to avoid hanging indefinitely.
func CheckInternetHTTP() (bool, error) {
	// Use a short timeout to prevent hanging
	client := http.Client{
		// Half second
		Timeout: 500 * time.Millisecond, // Adjust timeout as needed
	}

	// Try reaching Google's generate_204 endpoint, known for reliability
	// You can also use "https://www.google.com/generate_204"
	resp, err := client.Get("http://clients3.google.com/generate_204")
	if err != nil {
		// Check if the error is network-related (optional, could be too broad)
		// if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		//  return false, fmt.Errorf("timeout checking internet connection: %w", err)
		// }
		// Consider any error here as a potential lack of connectivity
		return false, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check for the expected 204 No Content status code
	// If you use a different URL (like google.com), check for 200 OK
	if resp.StatusCode == http.StatusNoContent {
		return true, nil
	}

	// Unexpected status code might indicate an issue (like a captive portal)
	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

func strSliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true // Found!
		}
	}
	return false // Not found after checking all elements
}

func Fatalf(format string, a ...any) {
	fmt.Printf(format, a...)
	os.Exit(1)
}

func durationInSeconds(nano time.Duration) float64 {
	const nanoDivisor float64 = 1_000_000_000
	return float64(nano) / nanoDivisor
}

func GetPerplexityAPIKey() string {
	return os.Getenv(perplexityApiKey)
}

func GetGeminiAPIKey() string {
	return os.Getenv(GeminiApiKey)
}

func GetChatGPTAPIKey() string {
	return os.Getenv(chatGPTApiKey)
}

func PrintAPIKeys() {
	fmtStr := "\nPerplexity API key is: %+v\nChatGPT API key is: %+v\nGemini API key is: %+v\n"
	fmt.Printf(fmtStr, GetPerplexityAPIKey(), GetChatGPTAPIKey(), GetGeminiAPIKey())
}

func RenderWithGlamour(text string) {
	// Use Glamour for rendering                                                                                                                                // You can customize options like style (dark, light, notty), word wrap, etc.                                                                               // Default options:
	renderer, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(0))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating renderer: %v\n", err)
		os.Exit(1)
	}

	out, err := renderer.Render(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering markdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(out)
}

func PrintUsage(connectedToInternet bool) {
	usageFmt := `%s:
	-c	use ChatGPT
	-g	use Gemini
	-lg	list Gemini models
	-p	use Perplexity
	-t	test API keys (note: they will be displayed)
	-h	show (this) help

	API keys should be set using the environment variables below:

	# For Perplexity
	export %s="your Perplexity API key here"

	# For ChatGPT
	export %s="your OpenAI API key here"

	# For Gemini
	export %s="your Gemini API key here"

`
	apiKeyExtendo := "\t - You already have %s set\n"

	haveGeminiAPIKey := GetGeminiAPIKey() != ""
	havePerplexityAPIKey := GetPerplexityAPIKey() != ""
	haveChatGPTAPIKey := GetChatGPTAPIKey() != ""

	if haveGeminiAPIKey || havePerplexityAPIKey {
		usageFmt += "\n\tSetup:\n"

		if havePerplexityAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, perplexityApiKey)
		}

		if haveChatGPTAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, chatGPTApiKey)
		}

		if haveGeminiAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, GeminiApiKey)
		}

	}

	if connectedToInternet {
		usageFmt += "\t - You are connected to the internet\n"
	}

	if haveChatGPTAPIKey && haveGeminiAPIKey && havePerplexityAPIKey && connectedToInternet {
		usageFmt += "\t - We're ready to rumble :)\n"
	}

	usageFmt += "\n"
	usage := fmt.Sprintf(usageFmt, os.Args[0], perplexityApiKey, chatGPTApiKey, GeminiApiKey)
	fmt.Print(usage)
}

func main() {
	listModelsToggle := false
	verboseToggle := true

	useGemini, usePerplexity, useChatGPT := false, false, false

	// Prints all arguments, including the program name
	// fmt.Println(os.Args)
	// fmt.Println("")

	// We do this here because we want the result in PrintUsage()
	connected, err := CheckInternetHTTP()

	for _, each := range os.Args {
		if strings.Contains(each, "-h") {
			PrintUsage(connected)
			os.Exit(1)
		}

		if strings.Contains(each, "-t") {
			PrintAPIKeys()
			os.Exit(0)
		}

		if strings.Contains(each, "-lg") {
			fmt.Println(GeminiMiddleWrapper("", true, true, false))
			os.Exit(0)
		}

		if strings.Contains(each, "-c") {
			useChatGPT = true
			fmt.Println("Using ChatGPT")
			break
		}

		if strings.Contains(each, "-g") {
			useGemini = true
			fmt.Println("Using Gemini")
			break
		}

		if strings.Contains(each, "-p") {
			usePerplexity = true
			fmt.Println("Using Perplexity")
			break
		}

	}

	// If none explicitly selected then use all
	if !(useChatGPT || useGemini || usePerplexity) {
		useChatGPT, useGemini, usePerplexity = true, true, true
	}

	if !connected {
		Fatalf("Not connected to the internet. Err is %v\n", err)
	}

	// Check we have API keys as required
	if useChatGPT && GetChatGPTAPIKey() == "" {
		Fatalf("Please set environment variable %s to use ChatGPT", chatGPTApiKey)
	}

	if useGemini && GetGeminiAPIKey() == "" {
		Fatalf("Please set environment variable %s to use Gemini", GeminiApiKey)
	}

	if usePerplexity && GetPerplexityAPIKey() == "" {
		Fatalf("Please set environment variable %s to use Perplexity", perplexityApiKey)
	}

	// --- 4. Read prompt from stdin ---
	reader := bufio.NewReader(os.Stdin)
	var promptText string
	var outputText string

	// Check if stdin is coming from a pipe or redirection
	fileInfo, _ := os.Stdin.Stat()
	isPipe := (fileInfo.Mode() & os.ModeCharDevice) == 0

	if !isPipe {
		// Interactive mode, display prompt
		fmt.Print("Prompt (press Ctrl+D when done) > ")
	}
	inputBytes, err := io.ReadAll(reader) // Read until EOF

	if err != nil {
		Fatalf("Failed to read input: %v", err)
	}
	// impliedly input is good

	promptText = strings.TrimSpace(string(inputBytes)) // Convert bytes to string

	if usePerplexity {
		fmt.Println("Hitting Perplexity API ...")
		outputText = PerplexityWrapper(promptText, false)
		RenderWithGlamour(outputText)
	}

	if useChatGPT {
		fmt.Println("Hitting ChatGPT API ...")
		outputText = ChatGPTWrapper(promptText, listModelsToggle, verboseToggle, false)
		RenderWithGlamour(outputText)
	}

	if useGemini {
		fmt.Println("Hitting Gemini API ...")
		outputText = GeminiWrapper(promptText, listModelsToggle, verboseToggle, false)
		RenderWithGlamour(outputText)
	}
}
