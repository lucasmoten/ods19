package server

import "net/http"

func (h *AppServer) homeListObjects(w http.ResponseWriter, r *http.Request, caller Caller) {

	parentID := r.URL.Query().Get("parentId")
	tmpl := h.TemplateCache.Lookup("listObjects.html")

	data := struct{ DistinguishedName, ParentID string }{
		DistinguishedName: caller.DistinguishedName,
		ParentID:          parentID,
	}

	tmpl.Execute(w, data)
}
