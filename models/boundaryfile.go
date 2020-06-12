package models

type BoundaryFileRequest struct {
	Query BoundaryFileQuery `json:"query"`
}

type BoundaryFileQuery struct {
	Term BoundaryFileTerm `json:"term"`
}

type BoundaryFileTerm struct {
	ID string `json:"id"`
}

// ------------------------------------------------------------------------

type BoundaryFileResponse struct {
	Hits EmbededHits `json:"hits"`
}
