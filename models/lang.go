package models

type Lang string

func (l Lang) String() string {
	return string(l)
}

const (
	LangFR Lang = "FR"
	LangEN Lang = "EN"
)
