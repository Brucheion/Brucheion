package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	//"strconv"
	"log"
	"strings"

	"golang.org/x/net/html"

	"github.com/ThomasK81/gocite"

	"github.com/gorilla/sessions" //for Cookiestore and other session functionality
)

//getSession will return an open session when there is a matching session by that name, and valid for the request.
//Note that it will also return a new session, if none was open by that name. -> Close the session after testing.
func getSession(req *http.Request) (*sessions.Session, error) {
	session, err := BrucheionStore.Get(req, SessionName)
	if err != nil {
		log.Printf("getSession: Error getting the session: %s\n", err)
		return nil, err
	}
	return session, nil
}

//getContent returns the response data from a GET request using url from parameter as a byte slice.
func getContent(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Read body: %v", err)
	}

	return data, nil
}

//getHref pulls the href attribute from a html Token
func getHref(token html.Token) (ok bool, href string) {
	for _, attribute := range token.Attr {
		if attribute.Key == "href" {
			href = attribute.Val
			ok = true
		}
	}
	return
}

//contains returns true if the 'needle' string is found in the 'heystack' string slice
func contains(heystack []string, needle string) bool {
	for _, straw := range heystack {
		if straw == needle {
			return true
		}
	}
	return false
}

//removeDuplicatesUnordered takes a string slice and returns it without any duplicates.
func removeDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
}

// extractLinks extracts the links from a gocite.Cite2Urn and returns them as a string slice
//called by newCollection
func extractLinks(urn gocite.Cite2Urn) (links []string, err error) {
	urnLink := urn.Namespace + "/" + strings.Replace(urn.Collection, ".", "/", -1) + "/"
	url := config.Host + "/static/image_archive/" + urnLink
	response, err := http.Get(url)
	if err != nil {
		return links, err
	}
	z := html.NewTokenizer(response.Body)
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}
			ok, url := getHref(t)
			if strings.Contains(url, ".dzi") {
				urnStr := urn.Base + ":" + urn.Protocol + ":" + urn.Namespace + ":" + urn.Collection + ":" + strings.Replace(url, ".dzi", "", -1)
				links = append(links, urnStr)
			}
			if !ok { //wouldn't this continue any way? Or put his before the if statement?
				continue
			}
		}
	}
	//return links, nil //go vet: unreacheable code: will return with err != nil or case tt anyway
}

//maxfloat returns the index of the highest float64 in a float64 slice (unused)
func maxfloat(floatslice []float64) int {
	max := floatslice[0]
	maxindex := 0
	for i, value := range floatslice {
		if value > max {
			max = value
			maxindex = i
		}
	}
	return maxindex
}

//testAllTheSame tests if all strings in a two-dimensional string-slice are the same (?)
func testAllTheSame(testset [][]string) bool {
	teststr := strings.Join(testset[0], "")
	for i := range testset {
		if i == 0 {
			continue
		}
		if teststr != strings.Join(testset[i], "") {
			return false
		}
	}
	return true
}

//findSpace returns a rune-slice exluding leading and trailing whitespaces
//(Works like strings.TrimSpace but for rune-slices)
//Additionally returns the count of trimmed leading and trailing whitespaces
//Used in addSansHyphens
func findSpace(runeSl []rune) (spBefore, spAfter int, newSl []rune) {
	spAfter = 0
	spBefore = 0
	for i := 0; i < len(runeSl); i++ {
		if runeSl[i] == rune(' ') {
			spBefore++ //continue
		} else {
			//since i is incremented anyway, wouldn't it be enough to set spBefore = i-1 here?
			break
		}
	}
	for i := len(runeSl) - 1; i >= 0; i-- {
		if runeSl[i] == rune(' ') {
			spAfter++
		} else {
			break
		}
	}
	return spBefore, spAfter, runeSl[spBefore : len(runeSl)-spAfter]
}

//testString does not seem to be in use anymore (?)
func testString(str string, strsl1 []string, cursorIn int) (cursorOut int, sl []int, ok bool) {
	calcStr1 := ""
	if len([]rune(str)) > len([]rune(strings.Join(strsl1[cursorIn:], ""))) {
		return 0, []int{}, false
	}
	base := cursorIn
	for i, v := range strsl1[cursorIn:] {
		calcStr1 = calcStr1 + v
		if calcStr1 != str {
			if i+1 == len(sl) {
				return 0, []int{}, false
			}
			sl = append(sl, i+base)
			continue
		}
		if calcStr1 == str {
			sl = append(sl, i+base)
			cursorOut = i + base + 1
			ok = true
			return cursorOut, sl, ok
		}
	}
	return 0, []int{}, false
}

//testStringSL is used for multipage alignment (?)
func testStringSl(slsl [][]string) (slsl2 [][][]int, ok bool) {
	if len(slsl) == 0 {
		// fmt.Println("zero length")
		slsl2 = [][][]int{}
		return slsl2, ok
	}
	ok = testAllTheSame(slsl)
	if !ok {
		// fmt.Println("slices not same length")
		slsl2 = [][][]int{}
		return slsl2, ok
	}
	// fmt.Println("passed testAllTheSame")

	length := len(slsl)

	base := make([]int, length)
	cursor := make([]int, length)
	indeces := make([][]int, length)
	testr := ""
	slsl2 = make([][][]int, length)

	for i := 0; i < len(slsl[0]); i++ {
		match := false
		indeces[0] = append(indeces[0], i)
		testr = testr + slsl[0][i]
		// fmt.Println("test", testr)
		// fmt.Scanln()

		for k := range slsl {
			if k == 0 {
				continue
			}
			cursor[k], indeces[k], match = testString(testr, slsl[k], base[k])
			if !match {
				// fmt.Println(testr, "and", slsl[k][base[k]:], "do not match")
				// fmt.Scanln()
				break
			}
		}
		if match {
			// fmt.Println("write to slice!!")
			// fmt.Scanln()
			for k := range slsl {
				slsl2[k] = append(slsl2[k], indeces[k])
				if k == 0 {
					continue
				}
				base[k] = cursor[k]
			}
			indeces[0] = []int{}
			testr = ""
		}
	}
	ok = true
	return slsl2, ok
}

//addSansHyphens adds Hyphens after certain sanscrit runes but not before
//used for nwa and multipage alignment
func addSansHyphens(s string) string {
	hyphen := []rune(`&shy;`)
	after := []rune{rune('a'), rune('ā'), rune('i'), rune('ī'), rune('u'), rune('ū'), rune('ṛ'), rune('ṝ'), rune('ḷ'), rune('ḹ'), rune('e'), rune('o'), rune('ṃ'), rune('ḥ')}
	notBefore := []rune{rune('ṃ'), rune('ḥ'), rune(' ')}
	runeSl := []rune(s)
	spBefore, spAfter, runeSl := findSpace(runeSl)
	newSl := []rune{}
	if len(runeSl) <= 2 {
		return s
	}
	newSl = append(newSl, runeSl[0:2]...)

	for i := 2; i < len(runeSl)-2; i++ {
		next := false
		possible := false
		for j := range after {
			if after[j] == runeSl[i] {
				possible = true
			}
		}
		if !possible {
			newSl = append(newSl, runeSl[i])
			continue
		}
		for j := range notBefore {
			if notBefore[j] == runeSl[i+1] {
				next = true
			}
		}
		if next {
			newSl = append(newSl, runeSl[i])
			//next = false //ineffectual assignment to next: next is not called after this if-clause and is set false with every run of the outer for-clause anyway.
			continue
		}
		if runeSl[i] == rune('a') {
			if runeSl[i+1] == rune('i') || runeSl[i+1] == rune('u') {
				newSl = append(newSl, runeSl[i])
				continue
			}
		}
		if runeSl[i-1] == rune(' ') {
			newSl = append(newSl, runeSl[i])
			continue
		}
		newSl = append(newSl, runeSl[i])
		for k := range hyphen {
			newSl = append(newSl, hyphen[k])
		}
	}
	SpBefore := []rune{}
	SpAfter := []rune{}
	for i := 0; i < spBefore; i++ {
		SpBefore = append(SpBefore, rune(' '))
	}
	for i := 0; i < spAfter; i++ {
		SpAfter = append(SpAfter, rune(' '))
	}
	if len(runeSl) < 4 {
		newSl = append(newSl, runeSl[len(runeSl)-1:]...)
	} else {
		newSl = append(newSl, runeSl[len(runeSl)-2:]...)
	}
	newSl = append(newSl, SpAfter...)
	newSl = append(SpBefore, newSl...)
	return string(newSl)
}
