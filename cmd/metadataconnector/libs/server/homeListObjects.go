package server

import (
	"net/http"

	cfg "decipher.com/oduploader/config"
)

func (h *AppServer) homeListObjects(w http.ResponseWriter, r *http.Request, caller Caller) {

	parentID := r.URL.Query().Get("parentId")
	tmpl := h.TemplateCache.Lookup("listObjects.html")

	data := struct{ RootURL, DistinguishedName, ParentID string }{
		RootURL:           cfg.RootURL,
		DistinguishedName: caller.DistinguishedName,
		ParentID:          parentID,
	}

	tmpl.Execute(w, data)
}
