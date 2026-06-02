#!/bin/bash

# Detect OS
OS="$(uname -s)"
case "$OS" in
	Linux*)    OS=Linux;;
	Darwin*)   OS=Darwin;;
	CYGWIN*)   OS=Cygwin;;
	MINGW*)    OS=MinGw;;
	*)         OS="UNKNOWN:${OS}"
esac

# Detect OpenCode installation
OPENCODE_INSTALLATION=""
if [ -d "~/.config/opencode" ]; then
	OPENCODE_INSTALLATION="~/.config/opencode"
elif command -v opencode &> /dev/null; then
	OPENCODE_INSTALLATION="$(which opencode)"
elif npm list -g opencode &> /dev/null; then
	OPENCODE_INSTALLATION="npm global"
fi

# Ask for install path or use default
INSTALL_PATH=""
if [ -z "$OPENCODE_INSTALLATION" ]; then
	read -p "Enter installation path [~/.config/opencode]: " INSTALL_PATH
	INSTALL_PATH=${INSTALL_PATH:-~/.config/opencode}
else
	INSTALL_PATH="$OPENCODE_INSTALLATION"
fi

# Build the Go binary or download pre-built
if [ "$OS" = "Linux" ] || [ "$OS" = "Darwin" ]; then
	if command -v go &> /dev/null; then
		go build -o "$INSTALL_PATH/opencode-kit" github.com/reeinharddd/okit/cmd/opencode-kit
	else
		curl -L https://github.com/reeinharddd/okit/releases/latest/download/opencode-kit-$OS -o "$INSTALL_PATH/opencode-kit"
		chmod +x "$INSTALL_PATH/opencode-kit"
	fi
else
	echo "Unsupported OS: $OS"
	exit 1
fi

# Initialize SQLite database
if [ ! -f "$INSTALL_PATH/opencode-kit.db" ]; then
	"$INSTALL_PATH/opencode-kit" init-db
fi

# Detect environment variables for API keys
API_KEYS=("GROQ_API_KEY" "MISTRAL_API_KEY" "OPENAI_API_KEY" "ANTHROPIC_API_KEY" "COHERE_API_KEY" "CEREBRAS_API_KEY" "GOOGLE_API_KEY" "NVIDIA_API_KEY" "OPENROUTER_API_KEY")

for key in "${API_KEYS[@]}"; do
	if [ -z "${!key}" ]; then
		echo "$key not found in environment variables"
	else
		echo "$key found in environment variables"
	fi
	done

# Seed known providers into DB
"$INSTALL_PATH/opencode-kit" seed-providers

# Run initial discovery (if API keys found)
if [ -n "$GROQ_API_KEY" ] || [ -n "$MISTRAL_API_KEY" ] || [ -n "$OPENAI_API_KEY" ] || [ -n "$ANTHROPIC_API_KEY" ] || [ -n "$COHERE_API_KEY" ] || [ -n "$CEREBRAS_API_KEY" ] || [ -n "$GOOGLE_API_KEY" ] || [ -n "$NVIDIA_API_KEY" ] || [ -n "$OPENROUTER_API_KEY" ]; then
	"$INSTALL_PATH/opencode-kit" discover
fi

# Generate initial config
if [ ! -f "$INSTALL_PATH/config.json" ]; then
	cat <<EOF > "$INSTALL_PATH/config.json"
{
	"api_keys": {
		"groq": "$GROQ_API_KEY",
		"mistral": "$MISTRAL_API_KEY",
		"openai": "$OPENAI_API_KEY",
		"anthropic": "$ANTHROPIC_API_KEY",
		"cohere": "$COHERE_API_KEY",
		"cerebras": "$CEREBRAS_API_KEY",
		"google": "$GOOGLE_API_KEY",
		"nvidia": "$NVIDIA_API_KEY",
		"openrouter": "$OPENROUTER_API_KEY"
	}
}
EOF
fi

# Print summary with next steps
echo "Installation completed successfully."
echo "Next steps:"
echo "1. Add API keys to $INSTALL_PATH/config.json"
echo "2. Run '$INSTALL_PATH/opencode-kit discover' to discover models"
echo "3. Run '$INSTALL_PATH/opencode-kit heal' to check for issues and fix them"
echo "4. Run '$INSTALL_PATH/opencode-kit' to start the opencode-kit service"