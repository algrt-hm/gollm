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

Binaries for Mac, Windows and Linux are in the `/bin` folder

You run the `gollm` command followed by your question or instruction. For example:

```bash
gollm "What's the weather like in London?"
```

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
	-lg	list Gemini models
	-p	use Perplexity
	-t	test API keys (note: they will be displayed)
	-h	show (this) help

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

**Genini**

- For GEMINI_API_KEY see: https://aistudio.google.com/app/plan_information
- For usage of the API see: https://console.cloud.google.com/apis/api/generativelanguage.googleapis.com/metrics
