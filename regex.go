package main

import "regexp"

var (
	JsonFileRegex            *regexp.Regexp
	MessageCountExtractRegex *regexp.Regexp
	WarningCountExtractRegex *regexp.Regexp
	ErrorCountExtractRegex   *regexp.Regexp
)

func CompileRegex() error {
	if regexp, err := regexp.Compile(`.*\.json$`); err != nil {
		return err
	} else {
		JsonFileRegex = regexp
	}

	if regexp, err := regexp.Compile("There are ([0-9]+?) message"); err != nil {
		return err
	} else {
		MessageCountExtractRegex = regexp
	}

	if regexp, err := regexp.Compile("There are ([0-9]+?) warning"); err != nil {
		return err
	} else {
		WarningCountExtractRegex = regexp
	}

	if regexp, err := regexp.Compile("There are ([0-9]+?) error"); err != nil {
		return err
	} else {
		ErrorCountExtractRegex = regexp
	}

	return nil
}
