package main

type File struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type LanguagePost struct {
	Cmd   []string `json:"cmd"`
	Env   []string `json:"env"`
	Files []File   `json:"files"`
}

type LanguagePostResponse struct {
	Output []string `json:"output"`
}

type LanguagePageData struct {
	Language string
	Version  string
}

type IndexPageData struct {
}
