package server_test

import (
	"strings"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/utils"
)

func TestThisTestPasses(t *testing.T) {
	// Nothing to see here
	v := false
	if v {
		t.Fail()
	}
}

func TestCombineInterfaces(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	//verboseOutput := testing.Verbose()

	startInterface := `{"users":["Alice","Bob"],"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2"]}}}`
	startObj, err := utils.UnmarshalStringToInterface(startInterface)
	if err != nil {
		t.Logf("Error unmarshalling: %v", err)
		t.FailNow()
	}

	var interfacesToAdd []string
	interfacesToAdd = append(interfacesToAdd, `{"users":"Jane"}`)
	interfacesToAdd = append(interfacesToAdd, `{"users":"Bob"}`)
	interfacesToAdd = append(interfacesToAdd, `{"users":"Jane"}`)
	interfacesToAdd = append(interfacesToAdd, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G3"]}}}`)
	interfacesToAdd = append(interfacesToAdd, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G2"]}}}`)
	interfacesToAdd = append(interfacesToAdd, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G2","G1","G3"]}}}`)
	interfacesToAdd = append(interfacesToAdd, `{"projects":{"project2":{"disp_nm":"PROJECT.2","groups":["G1","G3"]}}}`)
	interfacesToAdd = append(interfacesToAdd, `{"projects":{"project3":{"disp_nm":"PROJECT.3"}}}`)
	interfacesToAdd = append(interfacesToAdd, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G4"]}}}`)
	interfacesToAdd = append(interfacesToAdd, `{"projects":{"project2":{"disp_nm":"PROJECT.2","groups":["G2"]}}}`)
	interfacesToAdd = append(interfacesToAdd, `{"projects":{"project3":{"disp_nm":"PROJECT.3","groups":[]}}}`)

	var expectedResults []string
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2"]}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2"]}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2"]}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2","G3"]}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2","G3"]}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2","G3"]}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2","G3"]},"project2":{"disp_nm":"PROJECT.2","groups":["G1","G3"]}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2","G3"]},"project2":{"disp_nm":"PROJECT.2","groups":["G1","G3"]},"project3":{"disp_nm":"PROJECT.3"}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2","G3","G4"]},"project2":{"disp_nm":"PROJECT.2","groups":["G1","G3"]},"project3":{"disp_nm":"PROJECT.3"}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2","G3","G4"]},"project2":{"disp_nm":"PROJECT.2","groups":["G1","G3","G2"]},"project3":{"disp_nm":"PROJECT.3"}},"users":["Alice","Bob","Jane"]}`)
	expectedResults = append(expectedResults, `{"projects":{"project1":{"disp_nm":"PROJECT.1","groups":["G1","G2","G3","G4"]},"project2":{"disp_nm":"PROJECT.2","groups":["G1","G3","G2"]},"project3":{"disp_nm":"PROJECT.3","groups":[]}},"users":["Alice","Bob","Jane"]}`)

	t.Logf("Starting with")
	t.Logf(startInterface)
	t.Logf("Normalized...")
	startNorm, err := utils.NormalizeMarshalledInterface(startInterface)
	t.Logf(startNorm)

	for idx, i2Add := range interfacesToAdd {
		t.Logf("------------------------------")
		t.Logf("Adding #%d", idx)
		t.Logf(i2Add)
		t.Logf("Result...")

		addObj, err := utils.UnmarshalStringToInterface(i2Add)
		if err != nil {
			t.Logf("Error unmarshalling: %v", err)
			t.FailNow()
		}

		resultObj := utils.CombineInterface(startObj, addObj)
		resultStr, err := utils.MarshalInterfaceToString(resultObj)
		resultNorm, err := utils.NormalizeMarshalledInterface(resultStr)
		t.Logf(resultNorm)

		if strings.Compare(resultNorm, expectedResults[idx]) != 0 {
			t.Logf("Expected...")
			t.Logf(expectedResults[idx])
			t.Fail()
		}

		startObj = resultObj
	}

}
