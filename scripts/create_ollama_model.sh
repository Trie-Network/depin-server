#!/bin/bash

# Check if both parameters are provided
if [ $# -ne 2 ]; then
  echo "Error: Missing arguments. Usage: $0 <model-file-path.gguf> <model-name>"
  exit 1
fi

MODEL_FILE="$1"
MODEL_NAME="$2"

# Check if the file ends with .gguf
if [[ ! "$MODEL_FILE" =~ \.gguf$ ]]; then
  echo "Error: File name must end with .gguf"
  exit 1
fi

# Check if the file exists (any path)
if [ ! -f "$MODEL_FILE" ]; then
  echo "Error: File '$MODEL_FILE' not found."
  exit 1
fi

# Validate that model name is alphanumeric only
if [[ ! "$MODEL_NAME" =~ ^[a-zA-Z0-9]+$ ]]; then
  echo "Error: Model name must be alphanumeric only (letters and numbers)."
  exit 1
fi

# Print input summary
echo "Model file: $MODEL_FILE"
echo "Model name: $MODEL_NAME"

# Prepare model directory
mkdir -p "models/$MODEL_NAME"

# Copy and rename file
DEST_FILE="models/$MODEL_NAME/$MODEL_NAME.gguf"
cp "$MODEL_FILE" "$DEST_FILE"
if [ $? -ne 0 ]; then
  echo "Error: Failed to copy the model file."
  exit 1
fi

# Move into model directory
cd "models/$MODEL_NAME" || exit 1

# Create Modelfile
echo "FROM ./$MODEL_NAME.gguf" > Modelfile

echo "Created model directory and prepared files successfully."

# Create Ollama model
ollama create $MODEL_NAME -f Modelfile
if [ $? -ne 0 ]; then
  echo "Error: Failed to create the model with Ollama."
  exit 1
fi

# Success message
echo "Model '$MODEL_NAME' created successfully."
echo "You can now use the model with Ollama."
echo "To use the model, run: ollama run $MODEL_NAME"
echo "To list all models, run: ollama list"
echo "To delete the model, run: ollama delete $MODEL_NAME"

echo "Script completed successfully."
