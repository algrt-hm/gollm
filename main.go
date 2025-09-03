package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
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

// struct for options
type optionsStruct struct {
	useGemini     bool
	usePerplexity bool
	useChatGPT    bool
	useCerebras   bool
	logToJsonl    bool
	quietMode     bool

	readLog    bool
	readLogIdx int

	printUsage       bool
	printAPIKeys     bool
	listGeminiModels bool
	listOpenAIModels bool

	// Our promptText text will go in here
	promptText string
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

var logPath string
var logPathToPrint string
var connected bool

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
	 - Cerebras: %s`

	return "\t" + fmt.Sprintf(s,
		DefaultModels.Perplexity,
		DefaultModels.ChatGPT,
		DefaultModels.Gemini,
		DefaultModels.Cerebras,
	) + "\n"
}

func PrintUsage(connectedToInternet bool, logPath string) {
	// The logging path is dynamic

	usageFmt := `%s [options] [model]

	options:
	-h	show (this) help
	-lg	list Gemini models
	-lc	list OpenAI models
	-t	test API keys (note: they will be displayed)
	-l	enable logging of model interactions to %s
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
	usage := fmt.Sprintf(usageFmt, os.Args[0], logPathToPrint, perplexityApiKey, chatGPTApiKey, geminiApiKey, cerebrasApiKey)
	fmt.Print(usage + fmtDefaultModels())
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

	if o.useCerebras || o.usePerplexity || o.useGemini || o.useChatGPT {
		sort.Strings(modelsNameSlice)
		outS := strings.Join(modelsNameSlice, ", ")
		Print("Using " + outS)
	}

	// Let the user know if we're logging
	if !o.quietMode {
		Print("Logging to " + logPathToPrint)
	}

	// Check we have API keys as required
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

	// --- Run API calls concurrently ---
	var wg sync.WaitGroup

	if o.usePerplexity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Print("Hitting Perplexity API ...")
			Render(PerplexityWrapper(o.promptText, false, o.logToJsonl, o.quietMode))
		}()
	}

	if o.useChatGPT {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Print("Hitting ChatGPT API ...")
			Render(ChatGPTWrapper(o.promptText, false, o.logToJsonl, o.quietMode))
		}()
	}

	if o.useGemini {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Print("Hitting Gemini API ...")
			Render(GeminiWrapper(o.promptText, false, o.logToJsonl, o.quietMode))
		}()
	}

	if o.useCerebras {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Print("Hitting Cerebras API ...")
			Render(CerebrasWrapper(o.promptText, false, o.logToJsonl, o.quietMode))
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

	// ReadLogIdx
	if o.readLog {
		ReadLogIdx(o.readLogIdx)
		os.Exit(0)
	}

	// PrintUsage
	if o.printUsage {
		PrintUsage(connected, logPath)
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

	// TODO: need to strip this
	n := len([]rune(o.promptText))

	// Ask for input if we don't have any from stdin
	if n == 0 {
		o.promptText = readStdin(true)
	}

	callModels(o)
}

func handleOpts(argv []string, argc int) optionsStruct {
	var opts optionsStruct

	opts.useGemini, opts.usePerplexity, opts.useChatGPT, opts.useCerebras = false, false, false, false
	opts.logToJsonl = false
	opts.readLog = false

	// loop through each arg
	for idx, each := range argv {
		if strings.Contains(each, "-rl") {
			opts.readLog = true
			// negative means print all
			var logIdx = -1
			// TODO: should look ahead and see if the next argument can be an integer
			// if it can be, that's our idx for ReadLogIdx

			// if there is a next arg
			if idx+1 < argc {
				// try to atoi the next arg
				intArg, err := strconv.Atoi(argv[idx+1])
				// if successful use it
				if err == nil {
					logIdx = intArg
				// otherwise we need to raise an error
				} else {
					Fatalf("Invalid log index: %s, error is %s", argv[idx+1], err.Error())
				}
			}
			opts.readLogIdx = logIdx
		}

		if strings.Contains(each, "-h") {
			opts.printUsage = true
		}

		if strings.Contains(each, "-t") {
			opts.printAPIKeys = true
		}

		if strings.Contains(each, "-lg") {
			opts.listGeminiModels = true
		}

		if strings.Contains(each, "-lc") {
			opts.listOpenAIModels = true
		}

		if strings.Contains(each, "-q") {
			opts.quietMode = true

			if opts.logToJsonl {
				opts.logToJsonl = false
			}
		}

		if strings.Contains(each, "-l") {
			if opts.quietMode {
				fmt.Fprintf(os.Stderr, "Ignoring logging arg as quiet mode activated\n")
			} else {
				opts.logToJsonl = true
			}
		}

		if strings.Contains(each, "-c") {
			opts.useChatGPT = true
			break
		}

		if strings.Contains(each, "-g") {
			opts.useGemini = true
			break
		}

		if strings.Contains(each, "-p") {
			opts.usePerplexity = true
			break
		}

		if strings.Contains(each, "-f") {
			opts.useCerebras = true
			break
		}

		// Cerebras and ChatGPT combo
		if strings.Contains(each, "-b") {
			opts.useCerebras = true
			opts.useChatGPT = true
			break
		}
	}

	// Functionality for including the prompt as an argument
	// If we specify a model OR no models are specified
	if (opts.useChatGPT || opts.useGemini || opts.usePerplexity || opts.useCerebras) || !(opts.useChatGPT || opts.useGemini || opts.usePerplexity || opts.useCerebras) {
		// and we have not set readLog
		if !opts.readLog {
			// take the last arg
			// i.e. argv[argc-1]
			// and see if there is the flag character "-" in it
			// if not we assume it is the prompt
			if !strings.Contains(argv[argc-1], "-") {
				opts.promptText = argv[argc-1]
			}
		}
	}

	// If none explicitly selected
	if !(opts.useChatGPT || opts.useGemini || opts.usePerplexity || opts.useCerebras) {
		// and we have not set readLog
		if !opts.readLog {
			// then use all models
			opts.useChatGPT, opts.useGemini, opts.usePerplexity, opts.useCerebras = true, true, true, true
		}
	}

	return opts
}

func init() {
	// Ditto
	logPath = getLogPath()

	// For tidiness we replace $HOME with ~ in logPath
	logPathToPrint = strings.Replace(logPath, getHomeDir(), "~", 1)

	// We do this here because we want the result in PrintUsage()
	temp, err := CheckInternetHTTP()

	if err != nil {
		Fatalf("Issue with checking internet connection. Err is %v\n", err)
	}

	// update global
	connected = temp
}

func main() {
	// handle args
	argc := len(os.Args)
	argv := os.Args
	opts := handleOpts(argv, argc)

	// various bits of functionality use the quietMode global so we update this here
	quietMode = opts.quietMode

	// get whatever is being piped in
	opts.promptText += readPipe()

	// main bit of functionality
	core(opts)
}
