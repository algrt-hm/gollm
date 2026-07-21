package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
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

// struct for options
type optionsStruct struct {
	useGemini      bool
	usePerplexity  bool
	useChatGPT     bool
	useCerebras    bool
	useClaude      bool
	useDeepseek    bool
	modelsImplied  bool // true when no model flags were given and all providers defaulted on
	bypassLLMProxy bool
	useLLMProxy    bool
	logToJsonl     bool
	quietMode      bool
	interactive    bool

	readLog    bool
	readLogIdx int

	printUsage          bool
	printAPIKeys        bool
	listGeminiModels    bool
	listOpenAIModels    bool
	listAnthropicModels bool

	// Our promptText text will go in here
	promptText string
}

// Globals with various environment variable names for API keys
const perplexityApiKey = "PERPLEXITY_API_KEY"
const geminiApiKey = "GEMINI_API_KEY"
const chatGPTApiKey = "OPENAI_API_KEY"
const cerebrasApiKey = "CEREBRAS_API_KEY"
const claudeApiKey = "ANTHROPIC_API_KEY"
const deepseekApiKey = "DEEPSEEK_API_KEY"

// Suppress output for tests
// ... can be nice in development to turn this off for the full output in tests
const testingSuppressOutput = true

// if quiet mode is enabled:
// - we turn off logging
// - we can use raw print not glamour
// - we don't need headers or footers
var quietMode bool = false

var logPath string
var logPathToPrint string
var outputRenderMu sync.Mutex

// So we can point this to a noop for testing
var RenderWithGlamourPtr = RenderWithGlamour

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

// isConnectedToInternet is only consulted when printing the usage screen,
// so the (up to 500ms) check is done lazily rather than on every invocation
func isConnectedToInternet() bool {
	ok, err := CheckInternetHTTP()
	return err == nil && ok
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
		RenderWithGlamourPtr(s)
	} else {
		fmt.Println(s)
	}
}

// printAndRenderAtomically ensures one provider's status and rendered output
// are emitted as a single contiguous block when running providers concurrently.
func printAndRenderAtomically(statusLine string, renderedOutput string) {
	outputRenderMu.Lock()
	defer outputRenderMu.Unlock()

	if statusLine != "" {
		Print(statusLine)
	}
	Render(renderedOutput)
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
	msg := fmt.Sprintf(format, a...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprint(os.Stderr, msg)
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

func getClaudeAPIKey() string {
	return os.Getenv(claudeApiKey)
}

func getDeepseekAPIKey() string {
	return os.Getenv(deepseekApiKey)
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

func GetClaudeAPIKeyOrBail() string {
	ret := getClaudeAPIKey()
	if ret == "" {
		Fatalf("%s is not set", claudeApiKey)
	}
	return ret
}

func GetDeepseekAPIKeyOrBail() string {
	ret := getDeepseekAPIKey()
	if ret == "" {
		Fatalf("%s is not set", deepseekApiKey)
	}
	return ret
}

// maskAPIKey hides all but the last four characters of an API key
// so -t output is safe to show on a shared screen
func maskAPIKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	runes := []rune(key)
	if len(runes) <= 4 {
		return "****"
	}
	return "****" + string(runes[len(runes)-4:])
}

func PrintAPIKeys() {
	fmtStr := "\nPerplexity API key is: %s\nChatGPT API key is: %s\nGemini API key is: %s\nCerebras API key is: %s\nClaude API key is: %s\nDeepSeek API key is: %s\n"
	fmt.Printf(fmtStr,
		maskAPIKey(getPerplexityAPIKey()),
		maskAPIKey(getChatGPTAPIKey()),
		maskAPIKey(getGeminiAPIKey()),
		maskAPIKey(getCerebrasAPIKey()),
		maskAPIKey(getClaudeAPIKey()),
		maskAPIKey(getDeepseekAPIKey()),
	)
}

func RenderWithGlamour(text string) {
	// Use Glamour for rendering
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

func fmtDefaultModels() string {
	s := `Default models:
- Perplexity: %s
- ChatGPT: %s
- Gemini: %s
- Cerebras: %s
- Claude: %s
- DeepSeek: %s`

	return fmt.Sprintf(s,
		DefaultModels.Perplexity,
		DefaultModels.ChatGPT,
		DefaultModels.Gemini,
		DefaultModels.Cerebras,
		DefaultModels.Claude,
		DefaultModels.Deepseek,
	) + "\n"
}

func PrintUsage(connectedToInternet bool, logPathDisplay string) string {
	// The logging path is dynamic

	usageFmt := `%s [options] [model]

options:
-h	show (this) help
-i	interactive chat mode (requires single model, auto-selects first available)
-x	bypass LLM Proxy (proxy is used automatically when LLM_PROXY_URL is set)
-lg	list Gemini models
-lc	list OpenAI models
-la	list Anthropic models
-t	test API keys (shown masked, last four characters only)
-l	enable logging (disabled automatically when routing through LLM Proxy)
	logs model interactions to %s
-q	quiet mode: turns off logging and all non-essential output
-rl	[index]	show the log index, or if an index is provided, show the LLM response

model:
-c	use ChatGPT
-g	use Gemini
-f	use Cerebras
-p	use Perplexity
-s	use Claude (Sonnet)
-d	use DeepSeek

API keys should be set using the environment variables below:

# For Perplexity
export %s="your Perplexity API key here"

# For ChatGPT
export %s="your OpenAI API key here"

# For Gemini
export %s="your Gemini API key here"

# For Cerebras
export %s="your Cerebras API key here"

# For Claude
export %s="your Anthropic API key here"

# For DeepSeek
export %s="your DeepSeek API key here"

# For LLM Proxy (proxy enabled automatically when set, -x to bypass)
export LLM_PROXY_URL="http://localhost:8000/v1"
`
	apiKeyExtendo := "- You already have %s set\n"

	haveGeminiAPIKey := getGeminiAPIKey() != ""
	havePerplexityAPIKey := getPerplexityAPIKey() != ""
	haveChatGPTAPIKey := getChatGPTAPIKey() != ""
	haveCerebrasAPIKey := getCerebrasAPIKey() != ""
	haveClaudeAPIKey := getClaudeAPIKey() != ""
	haveDeepseekAPIKey := getDeepseekAPIKey() != ""
	haveLLMProxyURL := os.Getenv(llmProxyURLEnvVar) != ""

	// If we have any of the keys
	if haveGeminiAPIKey || havePerplexityAPIKey || haveChatGPTAPIKey || haveCerebrasAPIKey || haveClaudeAPIKey || haveDeepseekAPIKey || haveLLMProxyURL {
		usageFmt += "\nSetup:\n"

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

		if haveClaudeAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, claudeApiKey)
		}

		if haveDeepseekAPIKey {
			usageFmt += fmt.Sprintf(apiKeyExtendo, deepseekApiKey)
		}

		if haveLLMProxyURL {
			usageFmt += fmt.Sprintf(apiKeyExtendo, llmProxyURLEnvVar)
		}

	}
	// TODO: should we do something if impliedly we have none?

	if connectedToInternet {
		usageFmt += "- You are connected to the internet\n"
	} else {
		usageFmt += "- You are offline (or internet connectivity could not be verified)\n"
	}

	if haveChatGPTAPIKey && haveGeminiAPIKey && havePerplexityAPIKey && haveCerebrasAPIKey && haveClaudeAPIKey && haveDeepseekAPIKey && connectedToInternet {
		usageFmt += "- We're ready to rumble :)\n"
	}

	usageFmt += "\n"

	// os.Args[0] may be a nonsensical if testing
	usage := fmt.Sprintf(usageFmt, "gollm", logPathDisplay, perplexityApiKey, chatGPTApiKey, geminiApiKey, cerebrasApiKey, claudeApiKey, deepseekApiKey)
	return usage + fmtDefaultModels()
}

func readStdin(showPrompt bool) string {
	reader := bufio.NewReader(os.Stdin)

	if showPrompt {
		// Interactive mode, display prompt
		fmt.Print("Prompt (press Ctrl+D when done) > ")
	}

	inputBytes, err := io.ReadAll(reader) // Read until EOF

	if err != nil {
		Fatalf("Failed to read input: %v", err)
	}
	// impliedly input is good

	return strings.TrimSpace(string(inputBytes)) // Convert bytes to string
}

func readPipe() string {
	// Check if stdin is coming from a pipe or redirection
	fileInfo, _ := os.Stdin.Stat()
	isPipe := (fileInfo.Mode() & os.ModeCharDevice) == 0

	// Empty string if there's nothing coming in through stdin
	if !isPipe {
		return ""
	}

	return readStdin(false)
}

func callModels(o optionsStruct) {

	// When no models were explicitly requested we default to all providers;
	// skip any without an API key rather than exiting (the proxy handles auth itself)
	if o.modelsImplied && !o.useLLMProxy {
		skip := func(name string) { fmt.Fprintf(os.Stderr, "Skipping %s (no API key set)\n", name) }

		if o.useChatGPT && getChatGPTAPIKey() == "" {
			o.useChatGPT = false
			skip("ChatGPT")
		}

		if o.useGemini && getGeminiAPIKey() == "" {
			o.useGemini = false
			skip("Gemini")
		}

		if o.usePerplexity && getPerplexityAPIKey() == "" {
			o.usePerplexity = false
			skip("Perplexity")
		}

		if o.useCerebras && getCerebrasAPIKey() == "" {
			o.useCerebras = false
			skip("Cerebras")
		}

		if o.useClaude && getClaudeAPIKey() == "" {
			o.useClaude = false
			skip("Claude")
		}

		if o.useDeepseek && getDeepseekAPIKey() == "" {
			o.useDeepseek = false
			skip("DeepSeek")
		}

		if !(o.useChatGPT || o.useGemini || o.usePerplexity || o.useCerebras || o.useClaude || o.useDeepseek) {
			Fatalf("No API keys set — set at least one of the environment variables listed in gollm -h")
		}
	}

	// Tell the user which models we're using
	var modelsNameSlice []string

	if o.useCerebras {
		modelsNameSlice = append(modelsNameSlice, "Cerebras")
	}

	if o.usePerplexity {
		modelsNameSlice = append(modelsNameSlice, "Perplexity")
	}

	if o.useGemini {
		modelsNameSlice = append(modelsNameSlice, "Gemini")
	}

	if o.useChatGPT {
		modelsNameSlice = append(modelsNameSlice, "ChatGPT")
	}

	if o.useClaude {
		modelsNameSlice = append(modelsNameSlice, "Claude")
	}

	if o.useDeepseek {
		modelsNameSlice = append(modelsNameSlice, "DeepSeek")
	}

	if o.useCerebras || o.usePerplexity || o.useGemini || o.useChatGPT || o.useClaude || o.useDeepseek {
		sort.Strings(modelsNameSlice)
		outS := strings.Join(modelsNameSlice, ", ")
		if o.useLLMProxy {
			Print("Using " + outS + " (via proxy)")
		} else {
			Print("Using " + outS)
		}
	}

	// Let the user know if we're logging
	if !o.quietMode && o.logToJsonl {
		Print("Logging to " + logPathToPrint)
	}

	// Check we have API keys as required (skip when using proxy — proxy handles auth)
	if !o.useLLMProxy {
		if o.useChatGPT && getChatGPTAPIKey() == "" {
			Fatalf("Please set environment variable %s to use ChatGPT", chatGPTApiKey)
		}

		if o.useGemini && getGeminiAPIKey() == "" {
			Fatalf("Please set environment variable %s to use Gemini", geminiApiKey)
		}

		if o.usePerplexity && getPerplexityAPIKey() == "" {
			Fatalf("Please set environment variable %s to use Perplexity", perplexityApiKey)
		}

		if o.useCerebras && getCerebrasAPIKey() == "" {
			Fatalf("Please set environment variable %s to use Cerebras", cerebrasApiKey)
		}

		if o.useClaude && getClaudeAPIKey() == "" {
			Fatalf("Please set environment variable %s to use Claude", claudeApiKey)
		}

		if o.useDeepseek && getDeepseekAPIKey() == "" {
			Fatalf("Please set environment variable %s to use DeepSeek", deepseekApiKey)
		}
	}

	// --- Run API calls concurrently ---
	var wg sync.WaitGroup

	if o.usePerplexity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var statusLine string
			var output string
			var err error
			if o.useLLMProxy {
				statusLine = "Hitting Perplexity API (via proxy) ..."
				output, err = LLMProxyWrapperForProvider("Perplexity", o.promptText, proxyModelName(ProviderPerplexity, DefaultModels.Perplexity), false, o.logToJsonl, o.quietMode)
			} else {
				statusLine = "Hitting Perplexity API ..."
				output, err = PerplexityWrapper(o.promptText, false, o.logToJsonl, o.quietMode)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Perplexity failed: %v\n", err)
				return
			}
			printAndRenderAtomically(statusLine, output)
		}()
	}

	if o.useChatGPT {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var statusLine string
			var output string
			var err error
			if o.useLLMProxy {
				statusLine = "Hitting ChatGPT API (via proxy) ..."
				output, err = LLMProxyWrapperForProvider("ChatGPT", o.promptText, proxyModelName(ProviderChatGPT, string(DefaultModels.ChatGPT)), false, o.logToJsonl, o.quietMode)
			} else {
				statusLine = "Hitting ChatGPT API ..."
				output, err = ChatGPTWrapper(o.promptText, false, o.logToJsonl, o.quietMode)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "ChatGPT failed: %v\n", err)
				return
			}
			printAndRenderAtomically(statusLine, output)
		}()
	}

	if o.useGemini {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var statusLine string
			var output string
			var err error
			if o.useLLMProxy {
				statusLine = "Hitting Gemini API (via proxy) ..."
				output, err = LLMProxyWrapperForProvider("Gemini", o.promptText, proxyModelName(ProviderGemini, DefaultModels.Gemini), false, o.logToJsonl, o.quietMode)
			} else {
				statusLine = "Hitting Gemini API ..."
				output, err = GeminiWrapper(o.promptText, false, o.logToJsonl, o.quietMode)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Gemini failed: %v\n", err)
				return
			}
			printAndRenderAtomically(statusLine, output)
		}()
	}

	if o.useCerebras {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var statusLine string
			var output string
			var err error
			if o.useLLMProxy {
				statusLine = "Hitting Cerebras API (via proxy) ..."
				output, err = LLMProxyWrapperForProvider("Cerebras", o.promptText, proxyModelName(ProviderCerebras, DefaultModels.Cerebras), false, o.logToJsonl, o.quietMode)
			} else {
				statusLine = "Hitting Cerebras API ..."
				output, err = CerebrasWrapper(o.promptText, false, o.logToJsonl, o.quietMode)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cerebras failed: %v\n", err)
				return
			}
			printAndRenderAtomically(statusLine, output)
		}()
	}

	if o.useClaude {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var statusLine string
			var output string
			var err error
			if o.useLLMProxy {
				statusLine = "Hitting Claude API (via proxy) ..."
				output, err = LLMProxyWrapperForProvider("Claude", o.promptText, proxyModelName(ProviderClaude, DefaultModels.Claude), false, o.logToJsonl, o.quietMode)
			} else {
				statusLine = "Hitting Claude API ..."
				output, err = ClaudeWrapper(o.promptText, false, o.logToJsonl, o.quietMode)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Claude failed: %v\n", err)
				return
			}
			printAndRenderAtomically(statusLine, output)
		}()
	}

	if o.useDeepseek {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var statusLine string
			var output string
			var err error
			if o.useLLMProxy {
				statusLine = "Hitting DeepSeek API (via proxy) ..."
				output, err = LLMProxyWrapperForProvider("DeepSeek", o.promptText, proxyModelName(ProviderDeepseek, DefaultModels.Deepseek), false, o.logToJsonl, o.quietMode)
			} else {
				statusLine = "Hitting DeepSeek API ..."
				output, err = DeepseekWrapper(o.promptText, false, o.logToJsonl, o.quietMode)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "DeepSeek failed: %v\n", err)
				return
			}
			printAndRenderAtomically(statusLine, output)
		}()
	}

	// Wait here ensures main doesn't exit before goroutines finish
	wg.Wait()

	if !o.quietMode {
		RenderWithGlamour("# Done\n")
	} else {
		fmt.Fprintf(os.Stderr, "\n(Not logged as quiet mode activated)\n")
	}
}

func core(o optionsStruct) {
	// Run any of the single option bits

	// Interactive mode
	if o.interactive {
		InteractiveSession(o)
		os.Exit(0)
	}

	// ReadLogIdx
	if o.readLog {
		ReadLogIdx(o.readLogIdx)
		os.Exit(0)
	}

	// PrintUsage
	if o.printUsage {
		fmt.Println(PrintUsage(isConnectedToInternet(), logPathToPrint))
		os.Exit(0)
	}

	// PrintAPIKeys
	if o.printAPIKeys {
		PrintAPIKeys()
		os.Exit(0)
	}

	// ListGeminiModels
	if o.listGeminiModels {
		fmt.Println(ListGeminiModels())
		os.Exit(0)
	}

	// ListOpenAIModels
	if o.listOpenAIModels {
		fmt.Println(ListOpenAIModels())
		os.Exit(0)
	}

	// ListAnthropicModels
	if o.listAnthropicModels {
		fmt.Println(ListAnthropicModels())
		os.Exit(0)
	}

	// strip of any whitespace before getting length
	o.promptText = strings.TrimSpace(o.promptText)
	n := len([]rune(o.promptText))

	// Ask for input if we don't have any from stdin
	if n == 0 {
		o.promptText = readStdin(true)
	}

	callModels(o)
}

func init() {
	logPath = getLogPath()

	// For tidiness we replace $HOME with ~ in logPath
	logPathToPrint = strings.Replace(logPath, getHomeDir(), "~", 1)
}

func main() {
	// handle args
	argc := len(os.Args)
	argv := os.Args

	// No arguments provided — show usage
	if argc <= 1 {
		fmt.Println(PrintUsage(isConnectedToInternet(), logPathToPrint))
		os.Exit(0)
	}

	opts, err := handleOpts(argv, argc)

	if err != nil {
		Fatalf("Issue reading arguments: %s", err)
	}

	// various bits of functionality use the quietMode global so we update this here
	quietMode = opts.quietMode

	// Proxy is on by default when LLM_PROXY_URL is set; -x bypasses it
	if opts.bypassLLMProxy {
		opts.useLLMProxy = false
		fmt.Fprintf(os.Stderr, "LLM Proxy bypassed (-x flag).\n")
	} else if os.Getenv(llmProxyURLEnvVar) != "" {
		if CheckLLMProxyHealth() {
			opts.useLLMProxy = true
			fmt.Fprintf(os.Stderr, "LLM Proxy detected at %s, routing through proxy.\n", getLLMProxyURL())
			if opts.logToJsonl {
				opts.logToJsonl = false
				fmt.Fprintf(os.Stderr, "Logging disabled while using LLM Proxy.\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "LLM Proxy at %s is unreachable, falling back to direct API calls.\n", getLLMProxyURL())
		}
	}

	// get whatever is being piped in (but not in interactive mode)
	// a prompt argument and piped content are joined with a newline
	if !opts.interactive {
		if piped := readPipe(); piped != "" {
			if opts.promptText != "" {
				opts.promptText += "\n" + piped
			} else {
				opts.promptText = piped
			}
		}
	}

	// main bit of functionality
	core(opts)
}
