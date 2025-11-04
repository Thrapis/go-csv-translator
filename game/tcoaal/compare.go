package tcoaal

import "tw-translator/extracting"

func AreSameReplica(d1, d2 *extracting.DataLine) bool {
	return d1.Tag == d2.Tag
}
