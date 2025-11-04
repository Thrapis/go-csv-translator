package translating

import "tw-translator/extracting"

type PartialStringAnalyse func(string) *PartialString

type PartialStringGetTypeString func(*PartialString) []*StringPart

type PartialStringString func(*PartialString) string

type AreSameReplica func(*extracting.DataLine, *extracting.DataLine) bool

type ParasiteReplica func(string, string) string
