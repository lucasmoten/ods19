package server

import "net/http"

func (h *AppServer) homeListObjects(w http.ResponseWriter, r *http.Request, caller Caller) {

	tmpl := h.TemplateCache.Lookup("listObjects.html")

	data := struct{ DistinguishedName string }{
		DistinguishedName: caller.DistinguishedName,
	}

	tmpl.Execute(w, data)
}
