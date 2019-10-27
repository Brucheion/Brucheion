package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ThomasK81/gonwr"
)

type Word struct {
	Appearance string
	ID         int
	Alignment  int
	Highlight  float32
}

func nwa(text, text2 string) []string {
	hashreg := regexp.MustCompile(`#+`)
	punctreg := regexp.MustCompile(`[^\p{L}\s#]+`)
	swirlreg := regexp.MustCompile(`{[^}]*}`)
	text = swirlreg.ReplaceAllString(text, "")
	text2 = swirlreg.ReplaceAllString(text2, "")
	start := `<div class="tile is-child" lnum="L1">`
	start2 := `<div class="tile is-child" lnum="L2">`
	end := `</div>`
	collection := []string{text, text2}
	for i := range collection {
		collection[i] = strings.ToLower(collection[i])
	}
	var basetext []Word
	var comparetext []Word
	var highlight float32

	runealn1, runealn2, _ := gonwr.Align([]rune(collection[0]), []rune(collection[1]), rune('#'), 1, -1, -1)
	aln1 := string(runealn1)
	aln2 := string(runealn2)
	aligncol := fieldNWA([]string{aln1, aln2})
	aligned1, aligned2 := aligncol[0], aligncol[1]
	for i := range aligned1 {
		tmpA := hashreg.ReplaceAllString(aligned1[i], "")
		tmpB := hashreg.ReplaceAllString(aligned2[i], "")
		tmp2A := punctreg.ReplaceAllString(tmpA, "")
		tmp2B := punctreg.ReplaceAllString(tmpB, "")
		_, _, score := gonwr.Align([]rune(tmp2A), []rune(tmp2B), rune('#'), 1, -1, -1)
		base := len([]rune(tmpA))
		if len([]rune(tmpB)) > base {
			base = len([]rune(tmpB))
		}
		switch {
		case score <= 0:
			highlight = 1.0
		case score >= base:
			highlight = 0.0
		default:
			highlight = 1.0 - float32(score)/float32(base)
		}
		basetext = append(basetext, Word{Appearance: tmpA, ID: i + 1, Alignment: i + 1, Highlight: highlight})
		comparetext = append(comparetext, Word{Appearance: tmpB, ID: i + 1, Alignment: i + 1, Highlight: highlight})

	}
	text2 = start2
	for i := range comparetext {
		s := fmt.Sprintf("%.2f", comparetext[i].Highlight)
		switch comparetext[i].ID {
		case 0:
			text2 = text2 + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" id=\"" + strconv.Itoa(i+1) + "\" alignment=\"" + strconv.Itoa(comparetext[i].Alignment) + "\">" + addSansHyphens(comparetext[i].Appearance) + "</span>" + " "
		default:
			text2 = text2 + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" id=\"" + strconv.Itoa(i+1) + "\" alignment=\"" + strconv.Itoa(comparetext[i].Alignment) + "\">" + addSansHyphens(comparetext[i].Appearance) + "</span>" + " "
		}
	}
	text2 = text2 + end

	text = start
	for i := range basetext {
		s := fmt.Sprintf("%.2f", basetext[i].Highlight)
		for j := range comparetext {
			if comparetext[j].Alignment == basetext[i].ID {
				basetext[i].Alignment = comparetext[j].ID
			}
		}
		text = text + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" + id=\"" + strconv.Itoa(basetext[i].ID) + "\" alignment=\"" + strconv.Itoa(basetext[i].Alignment) + "\">" + addSansHyphens(basetext[i].Appearance) + "</span>" + " "
	}
	text = text + end

	return []string{text, text2}
}

func nwa2(basetext, baseid string, texts, ids []string) (alignments Alignments) {
	hashreg := regexp.MustCompile(`#+`)
	punctreg := regexp.MustCompile(`[^\p{L}\s#]+`)
	swirlreg := regexp.MustCompile(`{[^}]*}`)
	var highlight float32

	for i := range texts {
		alignment := Alignment{}
		texts[i] = strings.ToLower(texts[i])
		texts[i] = strings.TrimSpace(texts[i])
		texts[i] = swirlreg.ReplaceAllString(texts[i], "")
		runealn1, runealn2, _ := gonwr.Align([]rune(basetext), []rune(texts[i]), rune('#'), 1, -1, -1)
		aln1 := string(runealn1)
		aln2 := string(runealn2)
		aligncol := fieldNWA2([]string{aln1, aln2})
		aligned1, aligned2 := aligncol[0], aligncol[1]
		for j := range aligned1 {
			tmpA := hashreg.ReplaceAllString(aligned1[j], "")
			tmpB := hashreg.ReplaceAllString(aligned2[j], "")
			tmp2A := punctreg.ReplaceAllString(tmpA, "")
			tmp2B := punctreg.ReplaceAllString(tmpB, "")
			_, _, score := gonwr.Align([]rune(tmp2A), []rune(tmp2B), rune('#'), 1, -1, -1)
			base := len([]rune(tmpA))
			if len([]rune(tmpB)) > base {
				base = len([]rune(tmpB))
			}
			switch {
			case score <= 0:
				highlight = 1.0
			case score >= base:
				highlight = 0.0
			default:
				highlight = 1.0 - float32(score)/float32(base)
			}
			alignment.Source = append(alignment.Source, tmpA)
			alignment.Target = append(alignment.Target, tmpB)
			alignment.Score = append(alignment.Score, highlight)
		}
		newID := ids[i]
		alignments.Name = append(alignments.Name, newID)
		alignments.Alignment = append(alignments.Alignment, alignment)
	}
	return alignments
}

func fieldNWA(alntext []string) [][]string {
	letters := [][]string{}
	for i := range alntext {
		charSl := strings.Split(alntext[i], "")
		letters = append(letters, charSl)
	}
	length := len(letters)
	fields := make([][]string, length)
	tmp := make([]string, length)
	for i := range letters[0] {
		allspace := true
		for j := range letters {
			tmp[j] = tmp[j] + letters[j][i]
			if letters[j][i] != " " {
				allspace = false
			}
		}
		if allspace {
			for j := range letters {
				fields[j] = append(fields[j], tmp[j])
				tmp[j] = ""
			}
		}
	}
	for j := range letters {
		fields[j] = append(fields[j], tmp[j])
	}
	return fields
}

func fieldNWA2(alntext []string) [][]string {
	letters := [][]string{}
	for i := range alntext {
		charSl := strings.Split(alntext[i], "")
		letters = append(letters, charSl)
	}
	length := len(letters)
	fields := make([][]string, length)
	tmp := make([]string, length)
	for i := range letters[0] {
		allspace := true
		for j := range letters {
			tmp[j] = tmp[j] + letters[j][i]
			if letters[j][i] != " " {
				allspace = false
			}
		}
		if allspace {
			for j := range letters {
				fields[j] = append(fields[j], tmp[j])
				tmp[j] = ""
			}
		}
	}
	for j := range letters {
		fields[j] = append(fields[j], tmp[j])
	}
	for i := range fields {
		fields[i][0] = strings.TrimLeft(fields[i][0], " ")
	}
	return fields
}
