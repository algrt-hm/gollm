package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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

// Globals with various environment variable names for API keys
const perplexityApiKey = "PERPLEXITY_API_KEY"
const geminiApiKey = "GEMINI_API_KEY"
const chatGPTApiKey = "OPENAI_API_KEY"
const cerebrasApiKey = "CEREBRAS_API_KEY"

// if quiet mode is enabled:
// - we turn off logging
// - we can use raw print not glamour
// - we don't need headers or footers
var quietMode bool = false

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

func Print(s string) (int, error) {
	// not quietMode is the default case
	if !quietMode {
		return fmt.Println(s)
	}
	return 0, nil
}

func Render(s string) {
	// not quietMode is the default case
	if !quietMode {
		RenderWithGlamour(s)
	} else {
		fmt.Println(s)
	}
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

func getPerplexityAPIKey() string {
	return os.Getenv(perplexityApiKey)
}

func getGeminiAPIKey() string {
	return os.Getenv(geminiApiKey)
}

func getChatGPTAPIKey() string {
	return os.Getenv(chatGPTApiKey)
}

func getCerebrasAPIKey() string {
	return os.Getenv(cerebrasApiKey)
}

func GetPerplexityAPIKeyOrBail() string {
	ret := getPerplexityAPIKey()
	if ret == "" {
		Fatalf("%s is not set", perplexityApiKey)
	}
	return ret
}

func GetGeminiAPIKeyOrBail() string {
	ret := getGeminiAPIKey()
	if ret == "" {
		Fatalf("%s is not set", geminiApiKey)
	}
	return ret
}

func GetChatGPTAPIKeyOrBail() string {
	ret := getChatGPTAPIKey()
	if ret == "" {
		Fatalf("%s is not set", chatGPTApiKey)
	}
	return ret
}

func GetCerebrasAPIKeyOrBail() string {
	ret := getCerebrasAPIKey()
	if ret == "" {
		Fatalf("%s is not set", cerebrasApiKey)
	}
	return ret
}

func PrintAPIKeys() {
	fmtStr := "\nPerplexity API key is: %+v\nChatGPT API key is: %+v\nGemini API key is: %+v\n"
	fmt.Printf(fmtStr, getPerplexityAPIKey(), getChatGPTAPIKey(), getGeminiAPIKey())
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
	usageFmt := `%s [options] [model]

	options:
	-h	show (this) help
	-lg	list Gemini models
	-t	test API keys (note: they will be displayed)
	-l	enable logging of model interactions to ~/gollm_logs.jsonl
	-q	quiet mode: turns off logging and all non-essential output
	-rl	[index]	show the log index, or if an index is provided, show the LLM response

	model:
	-c	use ChatGPT
	-g	use Gemini
	-f	use Cerebras
	-p	use Perplexity

	API keys should be set using the environment variables below:

	# For Perplexity
	export %s="your Perplexity API key here"

	# For ChatGPT
	export %s="your OpenAI API key here"

	# For Gemini
	export %s="your Gemini API key here"

	# For Cerebras
	export %s="your Cerebras API key here"

`
	apiKeyExtendo := "\t - You already have %s set\n"

	haveGeminiAPIKey := getGeminiAPIKey() != ""
	havePerplexityAPIKey := getPerplexityAPIKey() != ""
	haveChatGPTAPIKey := getChatGPTAPIKey() != ""
	haveCerebrasAPIKey := getCerebrasAPIKey() != ""

	// If we have any of the keys
	if haveGeminiAPIKey || havePerplexityAPIKey || haveChatGPTAPIKey || haveCerebrasAPIKey {
		usageFmt += "\n\tSetup:\n"

		if havePerplexityAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, perplexityApiKey)
		}

		if haveChatGPTAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, chatGPTApiKey)
		}

		if haveGeminiAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, geminiApiKey)
		}

		if haveCerebrasAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, cerebrasApiKey)
		}

	}
	// TODO: should we do something if impliedly we have none?

	if connectedToInternet {
		usageFmt += "\t - You are connected to the internet\n"
	}

	if haveChatGPTAPIKey && haveGeminiAPIKey && havePerplexityAPIKey && haveCerebrasAPIKey && connectedToInternet {
		usageFmt += "\t - We're ready to rumble :)\n"
	}

	usageFmt += "\n"
	usage := fmt.Sprintf(usageFmt, os.Args[0], perplexityApiKey, chatGPTApiKey, geminiApiKey, cerebrasApiKey)
	fmt.Print(usage)
}

func main() {
	useGemini, usePerplexity, useChatGPT, useCerebras := false, false, false, false
	logToJsonl := false

	// We do this here because we want the result in PrintUsage()
	connected, err := CheckInternetHTTP()
	argc := len(os.Args)

	for idx, each := range os.Args {
		if strings.Contains(each, "-rl") {
			// negative means print all
			var logIdx = -1
			// TODO: should look ahead and see if the next argument can be an integer
			// if it can be, that's our idx for ReadLogIdx

			// if there is a next arg
			if idx+1 < argc {
				// try to atoi the next arg
				intArg, err := strconv.Atoi(os.Args[idx+1])
				// if successful use it
				if err == nil {
					logIdx = intArg
				}
			}

			ReadLogIdx(logIdx)
			os.Exit(0)
		}

		if strings.Contains(each, "-h") {
			PrintUsage(connected)
			os.Exit(0)
		}

		if strings.Contains(each, "-t") {
			PrintAPIKeys()
			os.Exit(0)
		}

		if strings.Contains(each, "-lg") {
			fmt.Println(ListGeminiModels())
			os.Exit(0)
		}

		if strings.Contains(each, "-lc") {
			fmt.Println(ListOpenAIModels())
			os.Exit(0)
		}

		if strings.Contains(each, "-q") {
			quietMode = true

			if logToJsonl {
				logToJsonl = false
				fmt.Fprintf(os.Stderr, "Not logging as quiet mode activated\n")
			}
		}

		if strings.Contains(each, "-l") {
			if quietMode {
				fmt.Fprintf(os.Stderr, "Not logging as quiet mode activated\n")
			} else {
				logToJsonl = true
			}
		}

		if strings.Contains(each, "-c") {
			useChatGPT = true
			Print("Using ChatGPT")
			break
		}

		if strings.Contains(each, "-g") {
			useGemini = true
			Print("Using Gemini")
			break
		}

		if strings.Contains(each, "-p") {
			usePerplexity = true
			Print("Using Perplexity")
			break
		}

		if strings.Contains(each, "-f") {
			useCerebras = true
			Print("Using Cerebras")
			break
		}
	}

	// Let the user know if we're logging
	Print("Logging")

	// If none explicitly selected then use all
	if !(useChatGPT || useGemini || usePerplexity || useCerebras) {
		useChatGPT, useGemini, usePerplexity, useCerebras = true, true, true, true
	}

	if !connected {
		Fatalf("Not connected to the internet. Err is %v\n", err)
	}

	// Check we have API keys as required
	if useChatGPT && getChatGPTAPIKey() == "" {
		Fatalf("Please set environment variable %s to use ChatGPT", chatGPTApiKey)
	}

	if useGemini && getGeminiAPIKey() == "" {
		Fatalf("Please set environment variable %s to use Gemini", geminiApiKey)
	}

	if usePerplexity && getPerplexityAPIKey() == "" {
		Fatalf("Please set environment variable %s to use Perplexity", perplexityApiKey)
	}

	if useCerebras && getCerebrasAPIKey() == "" {
		Fatalf("Please set environment variable %s to use Cerebras", cerebrasApiKey)
	}

	// --- Read prompt from stdin ---
	reader := bufio.NewReader(os.Stdin)
	var promptText string

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

	// --- Run API calls concurrently ---
	var wg sync.WaitGroup

	if usePerplexity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Print("Hitting Perplexity API ...")
			Render(PerplexityWrapper(promptText, false, logToJsonl, quietMode))
		}()
	}

	if useChatGPT {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Print("Hitting ChatGPT API ...")
			Render(ChatGPTWrapper(promptText, false, logToJsonl, quietMode))
		}()
	}

	if useGemini {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Print("Hitting Gemini API ...")
			Render(GeminiWrapper(promptText, false, logToJsonl, quietMode))
		}()
	}

	if useCerebras {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Print("Hitting Cerebras API ...")
			Render(CerebrasWrapper(promptText, false, logToJsonl, quietMode))
		}()
	}

	// Wait here ensures main doesn't exit before goroutines finish
	wg.Wait()

	if !quietMode {
		RenderWithGlamour("\n# Done\n")
	}
}
