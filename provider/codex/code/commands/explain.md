---
name: explain
description: Explain code, errors, or stack traces in context
---

# Explain

This command provides context-aware explanations for code, errors, and stack traces.

## Usage

- `/core:explain file.php:45` - Explain code at a specific line.
- `/core:explain error "error message"` - Explain a given error.
- `/core:explain stack "stack trace"` - Explain a given stack trace.

## Code Explanation (`file:line`)

When a file path and line number are provided, follow these steps:

1.  **Read the file**: Read the contents of the specified file.
2.  **Extract context**: Extract a few lines of code before and after the specified line number to understand the context.
3.  **Analyze the code**: Analyze the extracted code block to understand its purpose and functionality.
4.  **Provide an explanation**: Provide a clear and concise explanation of the code, including its role in the overall application.

## Error Explanation (`error`)

When an error message is provided, follow these- steps:

1.  **Analyze the error**: Parse the error message to identify the key components, such as the error type and location.
2.  **Identify the cause**: Based on the error message and your understanding of the codebase, determine the root cause of the error.
3.  **Suggest a fix**: Provide a clear and actionable fix for the error, including code snippets where appropriate.
4.  **Link to documentation**: If applicable, provide links to relevant documentation that can help the user understand the error and the suggested fix.

## Stack Trace Explanation (`stack`)

When a stack trace is provided, follow these steps:

1.  **Parse the stack trace**: Break down the stack trace into individual function calls, including the file path and line number for each call.
2.  **Analyze the call stack**: Analyze the sequence of calls to understand the execution flow that led to the current state.
3.  **Identify the origin**: Pinpoint the origin of the error or the relevant section of the stack trace.
4.  **Provide an explanation**: Explain the sequence of events in the stack trace in a clear and understandable way.
