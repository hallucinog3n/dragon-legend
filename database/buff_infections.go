package database

import (
	"database/sql"
	"fmt"

	gorp "gopkg.in/gorp.v1"
)

var (
	BuffInfections = make(map[int]*BuffInfection)
)

type BuffInfection struct {
	ID                int    `db:"id"`
	Name              string `db:"name"`
	PoisonDef         int    `db:"poison_def"`
	ParalysisDef      int    `db:"paralysis_def"`
	ConfusionDef      int    `db:"confusion_def"`
	BaseDef           int    `db:"base_def"`
	AdditionalDEF     int    `db:"additional_def"`
	ArtsDEF           int    `db:"arts_def"`
	AdditionalArtsDEF int    `db:"additional_arts_def"`
	MaxHP             int    `db:"max_hp"`
	HPRecoveryRate    int    `db:"hp_recovery_rate"`
	STR               int    `db:"str"`
	DEX               int    `db:"dex"`
	INT               int    `db:"int"`
	BaseHP            int    `db:"base_hp"`
	AdditionalHP      int    `db:"additional_hp"`
	BaseATK           int    `db:"base_atk"`
	AdditionalATK     int    `db:"additional_atk"`
	BaseArtsATK       int    `db:"base_arts_atk"`
	AdditionalArtsATK int    `db:"additional_arts_atk"`
}

func (e *BuffInfection) Create() error {
	return db.Insert(e)
}

func (e *BuffInfection) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *BuffInfection) Delete() error {
	_, err := db.Delete(e)
	return err
}

func (e *BuffInfection) Update() error {
	_, err := db.Update(e)
	return err
}

func getBuffInfections() error {
	var buffInfections []*BuffInfection
	query := `select * from data.buff_infections`

	if _, err := db.Select(&buffInfections, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getBuffInfections: %s", err.Error())
	}

	for _, b := range buffInfections {
		BuffInfections[b.ID] = b
	}

	return nil
}
