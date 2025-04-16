const scheme = String.raw;

export const VarCall = scheme`
  (call_expression
    (function_name) @fn
    (arguments (plain_value) @tokenName) @args
    (#eq? @fn "var")) @call
`;

export const VarCallWithFallback = scheme`
  (call_expression
    (function_name) @fn
    (arguments
      (plain_value) @tokenName
      (_) @fallback)
    (#eq? @fn "var")
    (#match? @fallback ".+"))
`;
