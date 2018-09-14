package models_test

import (
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
)

func TestAACFlatten(t *testing.T) {

	res := models.AACFlatten("CNDAOTESTtesttester01OU_S_GovernmentOUchimeraOUDAEOUPeopleCUS")

	if res != "cndaotesttesttester01ou_s_governmentouchimeraoudaeoupeoplecus" {
		t.Errorf("Result from models.AACFlatten was not lowercase %s", res)
	}

}
