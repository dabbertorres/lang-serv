package main

type File struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type LanguagePost struct {
	Cmd   string `json:"cmd"`
	Files []File `json:"files"`
}

type LanguagePageData struct {
	Language string
	Version  string
}

type IndexPageData struct {
}
