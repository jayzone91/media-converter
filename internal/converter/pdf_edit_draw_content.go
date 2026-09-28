package converter

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func appendPDFPageContent(
	ctx *model.Context,
	page int,
	content []byte,
) error {
	pageDict, _, _, err :=
		ctx.PageDict(
			page,
			false,
		)
	if err != nil {
		return fmt.Errorf(
			"read page %d for drawing: %w",
			page,
			err,
		)
	}

	if pageDict == nil {
		return fmt.Errorf(
			"page %d not found",
			page,
		)
	}

	object, found :=
		pageDict.Find(
			"Contents",
		)

	if !found ||
		object == nil {
		return insertPDFPageContent(
			ctx,
			pageDict,
			page,
			content,
		)
	}

	if indirectRef, ok :=
		object.(types.IndirectRef); ok {
		return appendPDFIndirectContent(
			ctx,
			page,
			indirectRef,
			content,
		)
	}

	switch value :=
		object.(type) {
	case types.StreamDict:
		return appendPDFStreamContent(
			nil,
			&value,
			page,
			content,
			pageDict,
		)

	case types.Array:
		return appendPDFContentArray(
			ctx,
			pageDict,
			page,
			value,
			content,
		)

	default:
		return fmt.Errorf(
			"unsupported page %d contents type %T",
			page,
			object,
		)
	}
}

func appendPDFIndirectContent(
	ctx *model.Context,
	page int,
	reference types.IndirectRef,
	content []byte,
) error {
	objectNumber :=
		reference.ObjectNumber.Value()

	generationNumber :=
		reference.GenerationNumber.Value()

	entry, found :=
		ctx.FindTableEntry(
			objectNumber,
			generationNumber,
		)

	if !found ||
		entry == nil ||
		entry.Object == nil {
		return fmt.Errorf(
			"page %d content stream object %d not found",
			page,
			objectNumber,
		)
	}

	stream, ok :=
		entry.Object.(types.StreamDict)

	if !ok {
		return fmt.Errorf(
			"page %d content object %d is %T",
			page,
			objectNumber,
			entry.Object,
		)
	}

	return appendPDFStreamContent(
		entry,
		&stream,
		page,
		content,
		nil,
	)
}

func appendPDFContentArray(
	ctx *model.Context,
	pageDict types.Dict,
	page int,
	array types.Array,
	content []byte,
) error {
	if len(array) == 0 {
		return insertPDFPageContent(
			ctx,
			pageDict,
			page,
			content,
		)
	}

	last :=
		array[len(array)-1]

	if reference, ok :=
		last.(types.IndirectRef); ok {
		objectNumber :=
			reference.ObjectNumber.Value()

		generationNumber :=
			reference.GenerationNumber.Value()

		entry, found :=
			ctx.FindTableEntry(
				objectNumber,
				generationNumber,
			)

		if found &&
			entry != nil &&
			entry.Object != nil {
			if stream, ok :=
				entry.Object.(types.StreamDict); ok {
				return appendPDFStreamContent(
					entry,
					&stream,
					page,
					content,
					nil,
				)
			}
		}
	}

	return appendPDFNewContentStream(
		ctx,
		pageDict,
		page,
		array,
		content,
	)
}

func appendPDFStreamContent(
	entry *model.XRefTableEntry,
	stream *types.StreamDict,
	page int,
	content []byte,
	pageDict types.Dict,
) error {
	if stream == nil {
		return fmt.Errorf(
			"page %d content stream is missing",
			page,
		)
	}

	if err := stream.Decode(); err != nil {
		return fmt.Errorf(
			"decode page %d content stream: %w",
			page,
			err,
		)
	}

	stream.Content =
		append(
			stream.Content,
			'\n',
		)

	stream.Content =
		append(
			stream.Content,
			content...,
		)

	if err := stream.Encode(); err != nil {
		return fmt.Errorf(
			"encode page %d content stream: %w",
			page,
			err,
		)
	}

	if entry != nil {
		entry.Object =
			*stream

		return nil
	}

	if pageDict != nil {
		pageDict.Insert(
			"Contents",
			*stream,
		)

		return nil
	}

	return fmt.Errorf(
		"page %d content stream cannot be updated",
		page,
	)
}

func insertPDFPageContent(
	ctx *model.Context,
	pageDict types.Dict,
	page int,
	content []byte,
) error {
	stream, err :=
		ctx.NewStreamDictForBuf(
			content,
		)
	if err != nil {
		return fmt.Errorf(
			"create drawing stream for page %d: %w",
			page,
			err,
		)
	}

	if err := stream.Encode(); err != nil {
		return fmt.Errorf(
			"encode drawing stream for page %d: %w",
			page,
			err,
		)
	}

	reference, err :=
		ctx.IndRefForNewObject(
			*stream,
		)
	if err != nil {
		return fmt.Errorf(
			"store drawing stream for page %d: %w",
			page,
			err,
		)
	}

	pageDict.Insert(
		"Contents",
		*reference,
	)

	return nil
}

func appendPDFNewContentStream(
	ctx *model.Context,
	pageDict types.Dict,
	page int,
	array types.Array,
	content []byte,
) error {
	stream, err :=
		ctx.NewStreamDictForBuf(
			content,
		)
	if err != nil {
		return fmt.Errorf(
			"create drawing stream for page %d: %w",
			page,
			err,
		)
	}

	if err := stream.Encode(); err != nil {
		return fmt.Errorf(
			"encode drawing stream for page %d: %w",
			page,
			err,
		)
	}

	reference, err :=
		ctx.IndRefForNewObject(
			*stream,
		)
	if err != nil {
		return fmt.Errorf(
			"store drawing stream for page %d: %w",
			page,
			err,
		)
	}

	contents :=
		append(
			types.Array{},
			array...,
		)

	contents =
		append(
			contents,
			*reference,
		)

	pageDict.Insert(
		"Contents",
		contents,
	)

	return nil
}
