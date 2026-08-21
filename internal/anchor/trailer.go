package anchor

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// Git trailer keys carried on every commit made under a plan.
const (
	TrailerDirection     = "Festival-Direction"
	TrailerNormalization = "Festival-Normalization"
)

var (
	// ErrNoTrailer is returned when a message carries no direction trailers.
	ErrNoTrailer = errors.New("no direction trailers")
	// ErrMalformedTrailer is returned for a present but invalid trailer value.
	ErrMalformedTrailer = errors.New("malformed direction trailer")

	trailerLineRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*: .+$`)
)

// Trailers are the parsed direction trailers of one commit message.
type Trailers struct {
	DirectionHash        string
	NormalizationVersion int
}

// FormatTrailers renders the two trailer lines for r, newline-terminated.
func FormatTrailers(r direction.Result) string {
	return TrailerDirection + ": " + r.DirectionHash + "\n" +
		TrailerNormalization + ": " + strconv.Itoa(r.NormalizationVersion) + "\n"
}

// ParseTrailers reads the direction trailers from message, following git's
// rule that trailers live in the final paragraph. Other trailers are ignored.
func ParseTrailers(message string) (Trailers, error) {
	content, _ := splitCommentTail(message)
	paras := paragraphs(content)
	if len(paras) == 0 || !isTrailerBlock(paras[len(paras)-1]) {
		return Trailers{}, errs.Wrap("parse trailers", ErrNoTrailer)
	}
	var t Trailers
	var found bool
	for _, line := range paras[len(paras)-1] {
		key, value, _ := strings.Cut(line, ": ")
		switch key {
		case TrailerDirection:
			if !ValidHash(value) {
				return Trailers{}, errs.Wrap("parse trailers: "+TrailerDirection+" "+value, ErrMalformedTrailer)
			}
			t.DirectionHash, found = value, true
		case TrailerNormalization:
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return Trailers{}, errs.Wrap("parse trailers: "+TrailerNormalization+" "+value, ErrMalformedTrailer)
			}
			t.NormalizationVersion = n
		}
	}
	if !found {
		return Trailers{}, errs.Wrap("parse trailers", ErrNoTrailer)
	}
	if t.NormalizationVersion == 0 {
		return Trailers{}, errs.Wrap("parse trailers: "+TrailerNormalization+" missing", ErrMalformedTrailer)
	}
	return t, nil
}

// AppendTrailers returns message with the direction trailers for r present
// exactly once. Existing Festival-* lines are replaced, other trailers kept,
// git's trailing comment block left in place, the subject never touched.
// changed is false when the message already carried these values.
func AppendTrailers(message string, r direction.Result) (string, bool) {
	content, tail := splitCommentTail(message)
	paras := paragraphs(content)
	want := []string{
		TrailerDirection + ": " + r.DirectionHash,
		TrailerNormalization + ": " + strconv.Itoa(r.NormalizationVersion),
	}
	if n := len(paras); n > 0 && isTrailerBlock(paras[n-1]) {
		var kept []string
		for _, line := range paras[n-1] {
			key, _, _ := strings.Cut(line, ": ")
			if key != TrailerDirection && key != TrailerNormalization {
				kept = append(kept, line)
			}
		}
		paras[n-1] = append(kept, want...)
	} else {
		paras = append(paras, want)
	}
	var b strings.Builder
	for i, p := range paras {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(strings.Join(p, "\n"))
		b.WriteString("\n")
	}
	out := b.String()
	if tail != "" {
		out += "\n" + tail
	}
	return out, out != message
}

// paragraphs splits content on blank lines, dropping trailing whitespace.
func paragraphs(content string) [][]string {
	var out [][]string
	var cur []string
	for _, line := range strings.Split(strings.TrimRight(content, " \t\r\n"), "\n") {
		line = strings.TrimRight(line, " \t\r")
		if line == "" {
			if len(cur) > 0 {
				out = append(out, cur)
				cur = nil
			}
			continue
		}
		cur = append(cur, line)
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func isTrailerBlock(lines []string) bool {
	for _, l := range lines {
		if !trailerLineRE.MatchString(l) {
			return false
		}
	}
	return len(lines) > 0
}

// splitCommentTail separates git's trailing "# ..." comment block (present
// in the file a commit-msg hook sees) from the message proper.
func splitCommentTail(message string) (content, tail string) {
	lines := strings.Split(message, "\n")
	cut := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l == "" || strings.HasPrefix(l, "#") {
			if strings.HasPrefix(l, "#") {
				cut = i
			}
			continue
		}
		break
	}
	if cut == len(lines) {
		return message, ""
	}
	return strings.Join(lines[:cut], "\n"), strings.Join(lines[cut:], "\n")
}
