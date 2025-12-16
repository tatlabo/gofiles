package sequel

func UpdateKeywords() (sqlList []string) {
	const updateKeywords = `UPDATE files SET keywords = ( to_tsvector('polish', (translate ( (SELECT data->'name')::text, ',._-+', '     ') || ' ' || (SELECT(data->'ext')::text) ) ) ) WHERE keywords IS NULL;`

	sqlList = append(sqlList, updateKeywords)
	return
}
