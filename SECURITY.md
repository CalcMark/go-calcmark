# Security Policy

## Security Model

CalcMark is designed to be safe for evaluating user-provided calculation code. The interpreter implements multiple layers of protection against denial-of-service (DOS) attacks and malicious inputs.

## Input Limits

### File Size Limits

**CLI Tool** (`cmd/calcmark`):
- Maximum file size: **1 MB**
- Rationale: Prevents memory exhaustion from huge files

**Library Usage**:
- No built-in limits - users should implement appropriate limits for their use case
- Recommendation: 100KB for interactive editors, 1MB for batch processing

### Expression Complexity Limits

**Nesting Depth** (planned):
- Maximum expression depth: **100 levels**
- Example: `(((((...))))` with 100 nested parentheses

**Token Count** (planned):
- Maximum tokens per document: **10,000 tokens**
- Prevents "token bomb" attacks

### String Length Limits

**Identifiers**:
- Maximum length: **256 characters**
- Prevents extremely long variable names

**Number Literals**:
- Maximum length: **100 characters**
- Prevents pathological number parsing

## Denial of Service Protections

### Protection Mechanisms

1. **File Size Validation**: Reject files >1MB before processing
2. **Path Validation**: Block directory traversal (`..` in paths)
3. **Extension Validation**: Only `.cm` and `.calcmark` files
4. **Timeout Protection** (recommended): Set timeouts in production environments

### Recommended Timeouts

```go
// Example: Evaluation with timeout
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

doneChan := make(chan result)
go func() {
    res, err := calcmark.Eval(userInput)
    doneChan <- result{res, err}
}()

select {
case r := <-doneChan:
    // Process result
case <-ctx.Done():
    // Handle timeout
}
```

### Attack Vectors and Mitigations

| Attack | Mitigation |
|--------|------------|
| Huge file (e.g., 1GB) | File size limit (1MB for CLI) |
| Deep nesting `(((...)))` | Depth tracking (planned: 100 levels) |
| Token bomb `x1+x2+x3+...` | Token count limit (planned: 10K) |
| Infinite loop in calculation | No loops in language (by design) |
| Regex DOS | No regex in language (by design) |
| Memory exhaustion | Limits on all inputs |

## Security Best Practices

### For Library Users

1. **Validate Input Size**: Check input length before parsing
   ```go
   const maxInputSize = 100 * 1024 // 100KB
   if len(userInput) > maxInputSize {
       return errors.New("input too large")
   }
   ```

2. **Set Timeouts**: Use context.WithTimeout for production
3. **Isolate Evaluations**: Use separate environments per user/session
4. **Sanitize File Paths**: Validate paths before file operations
5. **Rate Limit**: Implement rate limiting for public APIs

### For Application Developers

1. **Run in Sandbox**: Consider running CalcMark in isolated environments
2. **Resource Limits**: Use cgroups or containers to limit CPU/memory
3. **Monitor Usage**: Track evaluation times and resource usage
4. **Log Errors**: Log validation failures for security monitoring

## Reporting Security Issues

If you discover a security vulnerability in CalcMark:

1. **Do NOT** open a public GitHub issue
2. Email: [security contact - TBD]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

We will respond within 48 hours and provide updates as we investigate.

## Security Roadmap

Planned security enhancements:

- [ ] Implement nesting depth limits (v0.2.0)
- [ ] Implement token count limits (v0.2.0)
- [ ] Add security benchmarks (v0.2.0)
- [ ] Fuzzing infrastructure (v0.3.0)
- [ ] Security audit (v1.0.0)

## Known Limitations

### Not Sandboxed

CalcMark runs in the same process as your application. It does NOT provide:
- OS-level sandboxing
- Network isolation
- File system isolation

For high-security environments, run CalcMark in a container or VM.

### No Code Execution Prevention

CalcMark evaluates mathematical expressions. While it doesn't support:
- System calls
- File I/O
- Network access
- Code injection

...it can still consume CPU and memory. Apply appropriate limits.

## Compliance

### Data Privacy

CalcMark does NOT:
- Store user data
- Make network requests
- Access the file system (except when explicitly loading .cm files)
- Log sensitive information

### Third-Party Dependencies

CalcMark has minimal dependencies:
- `github.com/shopspring/decimal` - Arbitrary precision math
- Go standard library

All dependencies are audited for security issues.

## Version Support

- **Current version**: Security updates provided
- **Previous minor version**: Critical fixes only
- **Older versions**: No security support

Upgrade to the latest version for best security.

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://golang.org/doc/security)
- [CWE: Common Weakness Enumeration](https://cwe.mitre.org/)
