-- name: InsertExtractionLog :exec
insert into extraction_logs
    (request_id, schedule_id, provider, model, prompt_version, duration_ms, success,
     error_message, page_url, prompt_tokens, completion_tokens, missing_fields_count,
     prompt_text, response_text)
values
    (@request_id, nullif(@schedule_id::text, '')::uuid, @provider, @model, @prompt_version,
     @duration_ms, @success, @error_message, @page_url, @prompt_tokens, @completion_tokens,
     @missing_fields_count, @prompt_text, @response_text);
