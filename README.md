# gollm

## What is `gollm`?

Have you ever wanted to quickly ask an AI a question or get its help directly from your computer's command line, without opening a web browser?

gollm lets you do just that! It's a simple tool that connects your terminal to powerful AI models like Google's Gemini and Perplexity.

## Why use `gollm`?

*   *Quick Answers:* Get fast responses to your questions or prompts.
*   *Convenience:* Interact with AI without leaving your terminal.
*   *Scripting:* Integrate AI capabilities into your command-line workflows or scripts (if you're into that!).

Think of it as having a helpful AI assistant available right where you do your command-line work.

## How to use

Binaries
- Binaries for Mac, Windows and Linux are in the `/bin` folder
- Note that the MacOS binaries are the ones labelled darwin and are available for both Apple Silicon (`gollm-darwin-arm64`) and Intel architectures (`gollm-darwin-amd64`)
- If you want to create your own binaries, simply clone the repo and run `make build`.

You run the `gollm` command followed by your question or instruction. For example:

```bash
gollm "Please tell me a little about yourself"
```

The prompt will be sent to any LLMs you have API keys set up for and the responses will be printed as they come back.

Or you can send text from another command to gollm:

```bash
cat my_document.txt | gollm "Summarize this text"
```

(Note: You'll need to set it up first, which involves getting API keys from the AI providers.)

## Usage

```
gollm:
	-c	use ChatGPT
	-g	use Gemini
	-h	show (this) help
	-l	use Cerebras
	-lg	list Gemini models
	-p	use Perplexity
	-t	test API keys (note: they will be displayed)

	API keys should be set using the environment variables below:

	# For Perplexity
	export PERPLEXITY_API_KEY="your Perplexity API key here"

	# For ChatGPT
	export OPENAI_API_KEY="your OpenAI API key here"

	# For Gemini
	export GEMINI_API_KEY="your Gemini API key here"

	# For Cerebras
	export CEREBRAS_API_KEY="your Cerebras API key here"

```

## More bits

**Go**

For installation of latest go on Ubuntu see: https://algrt.hm/2024-09-29-recent-go-on-popos/

**Gemini**

- For GEMINI_API_KEY see: https://aistudio.google.com/app/plan_information
- For usage of the API see: https://console.cloud.google.com/apis/api/generativelanguage.googleapis.com/metrics

## FAQs

*How do I set environment variables in Windows?*

To make environment variables persist across sessions:

1. Open System Properties:
	* Press <kbd>Win</kbd> + <kbd>R</kbd>, type `sysdm.cpl`, and press Enter.
	* Go to the Advanced tab.
	* Click on Environment Variables...
2. Add/Edit Variables:
	* Under "User variables" (for your account) or "System variables" (for all users), click New..., enter a name and value, then click OK.
	* To edit an existing variable, select it and click Edit...
3. Apply Changes:
	* Click OK on all dialogs to apply changes.
