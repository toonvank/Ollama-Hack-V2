import tiktoken


def get_token_count(text: str, model: str = "gpt-4") -> int:
    """
    Get the number of tokens in a text string for a given model.
    """
    encoding = tiktoken.encoding_for_model(model)
    tokens = encoding.encode(text)
    return len(tokens)


if __name__ == "__main__":
    # Example usage
    text = "Hello, how are you?"
    token_count = get_token_count(text)
    print(f"Token count for '{text}': {token_count}")
