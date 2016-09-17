//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package moss

import (
	"io"
)

// An iteratorSingle implements the Iterator interface, and is an edge
// case optimization when there's only a single segment to iterate and
// there's no lower-level iterator.  In contrast to the main iterator
// implementation, iteratorSingle doesn't have any heap operations.
type iteratorSingle struct {
	s      *segment
	posEnd int // Found via endKeyExclusive, or the segment length.
	pos    int // Logical entry position into segment.

	op uint64
	k  []byte
	v  []byte

	closer io.Closer

	options *CollectionOptions
}

// Close must be invoked to release resources.
func (iter *iteratorSingle) Close() error {
	if iter.closer != nil {
		iter.closer.Close()
		iter.closer = nil
	}

	return nil
}

// Next returns ErrIteratorDone if the iterator is done.
func (iter *iteratorSingle) Next() error {
	iter.pos++
	if iter.pos >= iter.posEnd {
		return ErrIteratorDone
	}

	iter.op, iter.k, iter.v = iter.s.GetOperationKeyVal(iter.pos)

	return nil
}

// Current returns ErrIteratorDone if the iterator is done.
// Otherwise, Current() returns the current key and val, which should
// be treated as immutable or read-only.  The key and val bytes will
// remain available until the next call to Next() or Close().
func (iter *iteratorSingle) Current() ([]byte, []byte, error) {
	if iter.pos >= iter.posEnd {
		return nil, nil, ErrIteratorDone
	}

	if iter.op == OperationDel {
		return nil, nil, nil
	}

	if iter.op == OperationMerge {
		var mo MergeOperator
		if iter.options != nil {
			mo = iter.options.MergeOperator
		}
		if mo == nil {
			return iter.k, nil, ErrMergeOperatorNil
		}

		vMerged, ok := mo.FullMerge(iter.k, nil, [][]byte{iter.v})
		if !ok {
			return iter.k, nil, ErrMergeOperatorFullMergeFailed
		}

		return iter.k, vMerged, nil
	}

	return iter.k, iter.v, nil
}

// CurrentEx is a more advanced form of Current() that returns more
// metadata.  It returns ErrIteratorDone if the iterator is done.
// Otherwise, the current operation, key, val are returned.
func (iter *iteratorSingle) CurrentEx() (
	entryEx EntryEx, key, val []byte, err error) {
	if iter.pos >= iter.posEnd {
		return EntryEx{}, nil, nil, ErrIteratorDone
	}

	return EntryEx{Operation: iter.op}, iter.k, iter.v, nil
}
