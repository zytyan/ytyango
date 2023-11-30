package tgbotidparse

import (
	"encoding/binary"
)

type FileId struct {
	Version         int `default:"4"`
	SubVersion      int `default:"47"`
	DcId            int
	Type            int
	FileReference   string
	Url             string
	Id              int
	AccessHash      uint64
	VolumeId        int
	LocalId         int
	PhotoSizeSource PhotoSizeSourceI
}
type UnpackedFileId struct {
	data   []byte
	hasErr bool
	offset int
}

func (u *UnpackedFileId) checkRemain(size int) bool {
	// 防止panic
	if len(u.data[u.offset:]) < size {
		u.hasErr = true
		return false
	}
	return true
}
func (u *UnpackedFileId) readInt32() int {
	if u.checkRemain(4) {
		return 0
	}
	result := int(binary.LittleEndian.Uint32(u.data[u.offset:]))
	u.offset += 4
	return result
}
func (u *UnpackedFileId) readUInt64() uint64 {
	if u.checkRemain(8) {
		return 0
	}
	result := binary.LittleEndian.Uint64(u.data[u.offset:])
	u.offset += 8
	return result
}

func (u *UnpackedFileId) readByte() int {
	if u.checkRemain(1) {
		return 0
	}
	result := u.data[u.offset]
	u.offset++
	return int(result)
}
func (u *UnpackedFileId) readTlString() []byte {
	tlString, err, offset := unpackTLString(u.data[u.offset:])
	if err != nil {
		u.hasErr = true
		return nil
	}
	u.offset += offset
	return tlString
}

/*
if ($result['typeId'] <= PHOTO) {
        $parsePhotoSize = function () use (&$result, &$fileId) {
            $result['photosize_source'] = $result['subVersion'] >= 4 ? \unpack('V', \stream_get_contents($fileId, 4))[1] : 0;
            switch ($result['photosize_source']) {
                case PHOTOSIZE_SOURCE_LEGACY:
                    $result += \unpack(LONG.'secret', \stream_get_contents($fileId, 8));
                    fixLong($result, 'secret');
                    break;
                case PHOTOSIZE_SOURCE_THUMBNAIL:
                    $result += \unpack('Vfile_type/athumbnail_type', \stream_get_contents($fileId, 8));
                    break;
                case PHOTOSIZE_SOURCE_DIALOGPHOTO_BIG:
                case PHOTOSIZE_SOURCE_DIALOGPHOTO_SMALL:
                    $result['photo_size'] = $result['photosize_source'] === PHOTOSIZE_SOURCE_DIALOGPHOTO_SMALL ? 'photo_small' : 'photo_big';
                    $result['dialog_id'] = unpackLong(\stream_get_contents($fileId, 8));
                    $result['dialog_access_hash'] = \unpack(LONG, \stream_get_contents($fileId, 8))[1];
                    fixLong($result, 'dialog_access_hash');
                    break;
                case PHOTOSIZE_SOURCE_STICKERSET_THUMBNAIL:
                    $result += \unpack(LONG.'sticker_set_id/'.LONG.'sticker_set_access_hash', \stream_get_contents($fileId, 16));
                    fixLong($result, 'sticker_set_id');
                    fixLong($result, 'sticker_set_access_hash');
                    break;

                case PHOTOSIZE_SOURCE_FULL_LEGACY:
                    $result += \unpack(LONG.'volume_id/'.LONG.'secret/llocal_id', \stream_get_contents($fileId, 20));
                    fixLong($result, 'volume_id');
                    fixLong($result, 'secret');
                    break;
                case PHOTOSIZE_SOURCE_DIALOGPHOTO_BIG_LEGACY:
                case PHOTOSIZE_SOURCE_DIALOGPHOTO_SMALL_LEGACY:
                    $result['photo_size'] = $result['photosize_source'] === PHOTOSIZE_SOURCE_DIALOGPHOTO_SMALL_LEGACY ? 'photo_small' : 'photo_big';
                    $result['dialog_id'] = unpackLong(\stream_get_contents($fileId, 8));
                    $result['dialog_access_hash'] = \unpack(LONG, \stream_get_contents($fileId, 8))[1];
                    fixLong($result, 'dialog_access_hash');

                    $result += \unpack(LONG.'volume_id/llocal_id', \stream_get_contents($fileId, 12));
                    fixLong($result, 'volume_id');
                    break;
                case PHOTOSIZE_SOURCE_STICKERSET_THUMBNAIL_LEGACY:
                    $result += \unpack(LONG.'sticker_set_id/'.LONG.'sticker_set_access_hash', \stream_get_contents($fileId, 16));
                    fixLong($result, 'sticker_set_id');
                    fixLong($result, 'sticker_set_access_hash');

                    $result += \unpack(LONG.'volume_id/llocal_id', \stream_get_contents($fileId, 12));
                    fixLong($result, 'volume_id');
                    break;

                case PHOTOSIZE_SOURCE_STICKERSET_THUMBNAIL_VERSION:
                    $result += \unpack(LONG.'sticker_set_id/'.LONG.'sticker_set_access_hash/lsticker_version', \stream_get_contents($fileId, 20));
                    fixLong($result, 'sticker_set_id');
                    fixLong($result, 'sticker_set_access_hash');
                    break;
            }
        };
*/

func FromBotAPIFileId(fileIdStr string) (FileId, error) {
	id, err := decodeBotFileId(fileIdStr)
	var fileId FileId
	if err != nil {
		return fileId, err
	}
	tail := 0

	version := int(id[len(id)-1])
	tail++
	subVersion := 0
	if version >= 4 {
		subVersion = int(id[len(id)-2])
		tail++
	}
	sid := UnpackedFileId{
		data:   id[:len(id)-tail],
		hasErr: false,
		offset: 0,
	}
	fileId.Version, fileId.SubVersion = version, subVersion
	typeId := uint32(sid.readInt32())
	fileId.DcId = sid.readInt32()
	hasReference := typeId&FileReferenceFlag != 0
	hasWebLocation := typeId&WebLocationFlag != 0
	typeId &= ^uint32(FileReferenceFlag | WebLocationFlag)
	if hasReference {
		fileId.FileReference = string(sid.readTlString())
	}
	if hasWebLocation {
		fileId.Url = string(sid.readTlString())
		fileId.AccessHash = sid.readUInt64()
	}
	fileId.Type = int(typeId)
	//sizeSrcBase := NewPhotoSizeSource(int(typeId))
	//switch typeId {
	//case PhotoSizeSourceLegacyC:
	//	fileId.PhotoSizeSource = &PhotoSizeSourceLegacy{
	//		PhotoSizeSource: sizeSrcBase,
	//		Secret:          sid.readInt32(),
	//	}
	//case PhotoSizeSourceThumbnailC:
	//	fileId.PhotoSizeSource = &PhotoSizeSourceThumbnail{
	//		PhotoSizeSource: sizeSrcBase,
	//		ThumbFileType:   sid.readInt32(),
	//		ThumbType:       string(sid.readTlString()),
	//	}
	//case PhotoSizeSourceDialogPhotoBig, PhotoSizeSourceDialogPhotoSmall:
	//	fileId.PhotoSizeSource = &PhotoSizeSourceDialogPhoto{
	//		PhotoSizeSource:  sizeSrcBase,
	//		DialogId:         sid.readInt32(),
	//		DialogAccessHash: sid.readInt32(),
	//	}
	//case PhotoSizeSourceStickerSetThumbnail:
	//	fileId.PhotoSizeSource = &PhotoSizeSourceStickersetThumbnail{
	//		PhotoSizeSource:      sizeSrcBase,
	//		StickerSetId:         sid.readInt32(),
	//		StickerSetAccessHash: sid.readInt32(),
	//	}
	//case PhotoSizeSourceFullLegacy:
	//	fileId.PhotoSizeSource = &PhotoSizeSourceLegacy{
	//		PhotoSizeSource: sizeSrcBase,
	//		Secret:          sid.readInt32(),
	//		VolumeId:        sid.readInt32(),
	//		LocalId:         sid.readInt32(),
	//	}
	//case PhotoSizeSourceDialogPhotoBigLegacy, PhotoSizeSourceDialogPhotoSmallLegacy:
	//	fileId.PhotoSizeSource = &PhotoSizeSourceDialogPhoto{
	//		PhotoSizeSource:  sizeSrcBase,
	//		DialogId:         sid.readInt32(),
	//		DialogAccessHash: sid.readInt32(),
	//		VolumeId:         sid.readInt32(),
	//	}
	//case PhotoSizeSourceStickerSetThumbnailLegacy:
	//	fileId.PhotoSizeSource = &PhotoSizeSourceStickersetThumbnail{
	//		PhotoSizeSource:      sizeSrcBase,
	//		StickerSetId:         sid.readInt32(),
	//		StickerSetAccessHash: sid.readInt32(),
	//		VolumeId:             sid.readInt32(),
	//	}
	//case PhotoSizeSourceStickerSetThumbnailVersion:
	//	fileId.PhotoSizeSource = &PhotoSizeSourceStickersetThumbnailVersion{
	//		PhotoSizeSource:      sizeSrcBase,
	//		StickerSetId:         sid.readUInt64(),
	//		StickerSetAccessHash: sid.readUInt64(),
	//		StickerSetVersion:    sid.readInt32(),
	//	}
	//default:
	//	return fileId, errors.New("unknown type id")
	//}
	return fileId, nil
}

type PhotoSizeSourceI interface {
	GetType() int
}
type PhotoSizeSource struct {
	Type int
}

func (p *PhotoSizeSource) GetType() int {
	return p.Type
}
func NewPhotoSizeSource(t int) PhotoSizeSource {
	return PhotoSizeSource{Type: t}
}

type PhotoSizeSourceStickersetThumbnailVersion struct {
	PhotoSizeSource
	StickerSetId         int
	StickerSetAccessHash int
	StickerSetVersion    int
}

type PhotoSizeSourceDialogPhoto struct {
	PhotoSizeSource
	DialogId         int
	DialogAccessHash int
}

type PhotoSizeSourceLegacy struct {
	PhotoSizeSource
	Secret int
}

type PhotoSizeSourceStickersetThumbnail struct {
	PhotoSizeSource
	StickerSetId         int
	StickerSetAccessHash int
}

type PhotoSizeSourceThumbnail struct {
	PhotoSizeSource
	ThumbFileType int
	ThumbType     string
}
