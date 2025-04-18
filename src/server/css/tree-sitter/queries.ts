const scheme = String.raw;

export const VarCall = scheme`
  (call_expression
    (function_name) @fn
    (arguments
      (plain_value) @tokenName) @arguments
    (#eq? @fn "var")) @call
`;

export const VarCallWithFallback = scheme`
  (call_expression
    (function_name) @fn
    (arguments
      (plain_value) @tokenName
      (_) @fallback) @arguments
    (#eq? @fn "var")
    (#match? @fallback ".+"))
`;

export const LightDarkValuesQuery = scheme`
  (call_expression
    (function_name) @fn
    (arguments
     (_) @lightValue
     (_) @darkValue)
    (#eq? @fn "light-dark"))
`
