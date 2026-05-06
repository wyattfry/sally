package web

import (
	"fmt"
	"html"
	"io"
	"net/http"

	queries "sally/server/internal/db/generated"
)

func (a app) uploadProjectThumbnail(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	_, project, ok := a.loadUserProject(w, r, projectID)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	url, err := saveUploadedFile(r, "thumbnail_file", a.uploadsDir)
	if err != nil {
		http.Error(w, "upload failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if url == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if _, err := a.queries.UpdateProject(r.Context(), queries.UpdateProjectParams{
		ID:           project.ID,
		Name:         project.Name,
		Address:      project.Address,
		Description:  project.Description,
		ThumbnailUrl: url,
	}); err != nil {
		http.Error(w, "could not update project", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	writeProjectHeroWrap(w, project.ID, url)
}

func (a app) uploadItemImage(w http.ResponseWriter, r *http.Request) {
	loaded, item, ok := a.loadProjectScheduleItem(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	url, err := saveUploadedFile(r, "source_image_file", a.uploadsDir)
	if err != nil {
		http.Error(w, "upload failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if url == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	_, err = a.queries.UpdateScheduleItem(r.Context(), queries.UpdateScheduleItemParams{
		ID:             item.ID,
		Data:           item.Data,
		Zone:           item.Zone,
		SourceUrl:      item.SourceUrl,
		SourceTitle:    item.SourceTitle,
		SourceImageUrl: url,
		SourcePdfLinks: item.SourcePdfLinks,
		Position:       item.Position,
	})
	if err != nil {
		http.Error(w, "could not update item", http.StatusInternalServerError)
		return
	}
	uploadURL := fmt.Sprintf("/projects/%s/schedules/%s/items/%s/thumbnail",
		loaded.project.ID, loaded.schedule.ID, item.ID)
	w.Header().Set("Content-Type", "text/html")
	writeItemThumbCell(w, item.ID, url, uploadURL)
}

// writeProjectHeroWrap writes the hero image fragment including the upload overlay.
// It is the swap target for project thumbnail uploads.
func writeProjectHeroWrap(w http.ResponseWriter, projectID, thumbnailURL string) {
	e := html.EscapeString
	id := "proj-hero-" + projectID
	uploadURL := "/projects/" + projectID + "/thumbnail"
	imgHTML := fmt.Sprintf(`<img class="project-hero-img" src="%s" alt="">`, e(thumbnailURL))
	fmt.Fprintf(w,
		`<div class="project-hero-wrap upload-target" id="%s" onclick="this.querySelector('input[type=file]').click()" title="Click to upload a new thumbnail">%s<div class="upload-overlay"><span>Upload</span></div><form hx-post="%s" hx-target="#%s" hx-swap="outerHTML" hx-encoding="multipart/form-data"><input type="file" name="thumbnail_file" accept="image/*" style="display:none" onchange="this.closest('form').requestSubmit()"></form></div>`,
		e(id), imgHTML, e(uploadURL), e(id))
}

// writeItemThumbCell writes the thumb <td> fragment including the upload overlay.
// It is the swap target for item image uploads, and also used by writeItemRow.
func writeItemThumbCell(w io.Writer, itemID, imageURL, uploadURL string) {
	e := html.EscapeString
	id := "item-thumb-" + itemID
	var imgHTML string
	if imageURL != "" {
		imgHTML = fmt.Sprintf(`<img class="item-thumb" src="%s" alt="">`, e(imageURL))
	} else {
		imgHTML = `<div class="item-thumb-placeholder"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/></svg></div>`
	}
	fmt.Fprintf(w,
		`<td class="item-thumb-cell upload-target" id="%s" onclick="this.querySelector('input[type=file]').click()" title="Click to upload an image">%s<div class="upload-overlay"><span>↑</span></div><form hx-post="%s" hx-target="#%s" hx-swap="outerHTML" hx-encoding="multipart/form-data"><input type="file" name="source_image_file" accept="image/*" style="display:none" onchange="this.closest('form').requestSubmit()"></form></td>`,
		e(id), imgHTML, e(uploadURL), e(id))
}
