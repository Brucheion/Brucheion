package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// Sort-Matrix Interface

type dataframe struct {
	Indices []int
	Values1 []string
	Values2 []string
}

func (m dataframe) Len() int           { return len(m.Indices) }
func (m dataframe) Less(i, j int) bool { return m.Indices[i] < m.Indices[j] }
func (m dataframe) Swap(i, j int) {
	m.Indices[i], m.Indices[j] = m.Indices[j], m.Indices[i]
	m.Values1[i], m.Values1[j] = m.Values1[j], m.Values1[i]
	m.Values2[i], m.Values2[j] = m.Values2[j], m.Values2[i]
}

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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
