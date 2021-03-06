package cmd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	xhttp "github.com/minio/radio/cmd/http"
)

// Returns a hexadecimal representation of time at the
// time response is sent to the client.
func mustGetRequestID(t time.Time) string {
	return fmt.Sprintf("%X", t.UnixNano())
}

// Write http common headers
func setCommonHeaders(w http.ResponseWriter) {
	w.Header().Set(xhttp.ServerInfo, "Radio/"+ReleaseTag)
	// Set `x-amz-bucket-region` only if region is set on the server
	// by default minio uses an empty region.
	if region := globalServerRegion; region != "" {
		w.Header().Set(xhttp.AmzBucketRegion, region)
	}
	w.Header().Set(xhttp.AcceptRanges, "bytes")

	// Remove sensitive information
	RemoveSensitiveHeaders(w.Header())
}

// Encodes the response headers into XML format.
func encodeResponse(response interface{}) []byte {
	var bytesBuffer bytes.Buffer
	bytesBuffer.WriteString(xml.Header)
	e := xml.NewEncoder(&bytesBuffer)
	e.Encode(response)
	return bytesBuffer.Bytes()
}

// Write object header
func setObjectHeaders(w http.ResponseWriter, objInfo ObjectInfo, rs *HTTPRangeSpec) (err error) {
	// set common headers
	setCommonHeaders(w)

	// Set last modified time.
	lastModified := objInfo.ModTime.UTC().Format(http.TimeFormat)
	w.Header().Set(xhttp.LastModified, lastModified)

	// Set Etag if available.
	if objInfo.ETag != "" {
		w.Header()[xhttp.ETag] = []string{"\"" + objInfo.ETag + "\""}
	}

	if objInfo.ContentType != "" {
		w.Header().Set(xhttp.ContentType, objInfo.ContentType)
	}

	if objInfo.ContentEncoding != "" {
		w.Header().Set(xhttp.ContentEncoding, objInfo.ContentEncoding)
	}

	if !objInfo.Expires.IsZero() {
		w.Header().Set(xhttp.Expires, objInfo.Expires.UTC().Format(http.TimeFormat))
	}

	// Set all other user defined metadata.
	for k, v := range objInfo.UserDefined {
		if HasPrefix(k, ReservedMetadataPrefix) {
			// Do not need to send any internal metadata
			// values to client.
			continue
		}
		w.Header().Set(k, v)
	}

	totalObjectSize := objInfo.Size

	// for providing ranged content
	start, rangeLen, err := rs.GetOffsetLength(totalObjectSize)
	if err != nil {
		return err
	}

	// Set content length.
	w.Header().Set(xhttp.ContentLength, strconv.FormatInt(rangeLen, 10))
	if rs != nil {
		contentRange := fmt.Sprintf("bytes %d-%d/%d", start, start+rangeLen-1, totalObjectSize)
		w.Header().Set(xhttp.ContentRange, contentRange)
	}

	return nil
}
