package sqliteforest

import "database/sql"

// cursor is used for traversing trees.
type cursor struct {
	tx  *sql.Tx
	loc []byte
}

func (cr cursor) getRecord() (fullNode, error) {
	return rcGet(string(cr.loc), func() (fullNode, error) {
		var toret fullNode
		var lol []byte
		err := cr.tx.QueryRow("SELECT * FROM treenodes WHERE hash = $1", cr.loc).
			Scan(&lol, &toret.Key, &toret.Value, &toret.LeftHash, &toret.RightHash)
		return toret, err
	})
}

func (cr cursor) getLeft() (cursor, error) {
	rec, err := cr.getRecord()
	if err != nil {
		return cursor{}, err
	}
	return cursor{cr.tx, rec.LeftHash}, nil
}

func (cr cursor) getRight() (cursor, error) {
	rec, err := cr.getRecord()
	if err != nil {
		return cursor{}, err
	}
	return cursor{cr.tx, rec.RightHash}, nil
}

func (cr cursor) isNull() bool {
	return cr.loc == nil
}

func allocCursor(tx *sql.Tx, rec fullNode) (cursor, error) {
	// if the hash already exists, then don't touch anything
	var lol []byte
	err := tx.QueryRow("SELECT hash FROM treenodes WHERE hash = $1", rec.Hash()).Scan(&lol)
	if err == nil {
		return cursor{tx, rec.Hash()}, nil
	}
	_, err = tx.Exec("INSERT INTO treenodes VALUES ($1, $2, $3, $4, $5)",
		rec.Hash(), rec.Key, rec.Value, rec.LeftHash, rec.RightHash)
	if err != nil {
		return cursor{}, err
	}
	return cursor{tx, rec.Hash()}, nil
}
