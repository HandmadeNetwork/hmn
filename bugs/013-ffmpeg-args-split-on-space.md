# FFmpeg command built with `fmt.Sprintf` and `strings.Split(" ")`

**File:** `src/assets/assets.go:236-241`
**Severity:** Medium — breaks on any temp dir containing a space (common on Windows); future-fragile
**Status:** Confirmed

## The bug

```go
args := fmt.Sprintf("-i %s -filter_complex [0]select=gte(n\\,1)[s0] -map [s0] -c:v mjpeg -f mjpeg -vframes 1 pipe:1", file.Name())
if config.Config.PreviewGeneration.CPULimitPath != "" {
    args = fmt.Sprintf("-l 10 -- %s %s", execPath, args)
    execPath = config.Config.PreviewGeneration.CPULimitPath
}
ffmpegCmd := exec.CommandContext(ctx, execPath, strings.Split(args, " ")...)
```

`file.Name()` is the full path returned from `os.CreateTemp("", "hmnasset")`. On Windows, `$TEMP` is commonly under `C:\Users\<username>\AppData\Local\Temp`. If the username contains a space (or the operator has moved `$TEMP` to somewhere with a space — "My Games", "Program Files", etc.), the resulting path looks like:

```
C:\Users\Ben Visness\AppData\Local\Temp\hmnasset1234567890
```

`strings.Split(args, " ")` then produces `["-i", "C:\\Users\\Ben", "Visness\\AppData\\...", ...]` and ffmpeg gets two broken args instead of one filename. FFmpeg fails with "file not found" and every video preview silently goes unthumbnailed.

Linux with a Unicode-named home (uncommon) hits the same issue. CI environments that set `TMPDIR` to something with a space hit it too.

There is no shell involved (`exec.CommandContext` takes argv directly, not a command line), so splitting on space to simulate shell word-splitting is just reinventing a worse version of shell quoting.

## Fix

Build argv as a slice from the start. No splitting, no sprintf:

```go
args := []string{
    "-i", file.Name(),
    "-filter_complex", "[0]select=gte(n\\,1)[s0]",
    "-map", "[s0]",
    "-c:v", "mjpeg",
    "-f", "mjpeg",
    "-vframes", "1",
    "pipe:1",
}
if config.Config.PreviewGeneration.CPULimitPath != "" {
    args = append([]string{"-l", "10", "--", execPath}, args...)
    execPath = config.Config.PreviewGeneration.CPULimitPath
}
ffmpegCmd := exec.CommandContext(ctx, execPath, args...)
```

## Injection risk

Today, `file.Name()` is the *only* user-influenceable input, and it comes from `os.CreateTemp` whose template is hard-coded to `"hmnasset"`. An attacker cannot influence the temp filename, so there is no command-injection path — `exec.Command` doesn't spawn a shell anyway. The bug is purely functional. But the shape of the code is exactly what you'd grep for if you were hunting for injection, and the fix removes the smell.
