const scheme = String.raw;

export const VarCallWithOrWithoutFallback = scheme`
  (call_expression
    (function_name) @fn
    (arguments
      . (plain_value) @tokenName
      (_) * @fallback)
    (#eq? @fn "var")) @VarCallWithOrWithoutFallback
`;

export const VarCallNoFallback = scheme`
  (call_expression
    (function_name) @fn
    (arguments
      . (plain_value) @tokenName)
    (#eq? @fn "var")) @VarCallNoFallback
`;

export const VarCallWithFallback = scheme`
  (call_expression
    (function_name) @fn
    (arguments
      . (plain_value) @tokenName
      (_)+ @fallback)
    (#eq? @fn "var")) @VarCallWithFallback
`;

export const VarCallWithLightDarkFallback = scheme`
  (call_expression
    (function_name) @fn
    (arguments
     (_) @lightValue
     (_) @darkValue)
    (#eq? @fn "light-dark")) @VarCallWithLightDarkFallback
`;
