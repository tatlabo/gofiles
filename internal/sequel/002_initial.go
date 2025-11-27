package sequel

var updateKeywords = []string{
	`UPDATE files SET keywords = ( to_tsvector('polish', (translate ( (SELECT data->'name')::text, ',._-+', '     ') || ' ' || (SELECT(data->'ext')::text) ) ) ) WHERE keywords IS NULL;`,
}

func UpdateKeywords() []string {
	return updateKeywords
}
