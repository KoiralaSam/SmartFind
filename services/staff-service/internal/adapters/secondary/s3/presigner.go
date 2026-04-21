// Package s3media re-exports the shared S3 presigner so staff-service
// code that imports this path continues to compile unchanged.
package s3media

import s3shared "smartfind/shared/s3media"

type Presigner = s3shared.Presigner
type PresignedPut = s3shared.PresignedPut

var (
	LoadPresigner   = s3shared.LoadPresigner
	GetPresigner    = s3shared.GetPresigner
	ContentTypeToExt = s3shared.ContentTypeToExt
)
