package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"unsafe"

	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/macos/vision"
	"github.com/progrium/darwinkit/objc"
)

func main() {
	if err := realMain(os.Args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(args []string, w io.Writer) error {
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <image-path>\n", args[0])
		fmt.Fprintf(os.Stderr, "\nPerform OCR on an image file.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	fs.Parse(args[1:])

	if fs.NArg() != 1 {
		fs.Usage()
		return errors.New("expected exactly one image path")
	}

	path := fs.Arg(0)
	img, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var ocrErr error
	objc.WithAutoreleasePool(func() {
		req := vision.NewRecognizeTextRequest().Init()
		req.SetUsesLanguageCorrection(true)
		req.SetRecognitionLevel(vision.RequestTextRecognitionLevelAccurate)
		objc.Call[objc.Void](req, objc.Sel("setAutomaticallyDetectsLanguage:"), true)

		handler := vision.NewImageRequestHandler().InitWithDataOptions(img, nil)

		var errObj foundation.Error
		handler.PerformRequestsError([]vision.IRequest{req}, unsafe.Pointer(&errObj))
		if !errObj.IsNil() {
			ocrErr = errors.New(errObj.Description())
			return
		}

		for _, o := range req.Results() {
			observation := vision.RecognizedTextObservationFrom(o.Ptr())
			candidates := observation.TopCandidates(1)
			if len(candidates) > 0 {
				fmt.Fprintln(w, candidates[0].String())
			}
		}
	})

	return ocrErr
}
