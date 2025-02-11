package testhelper

import "github.com/yammerjp/cachembed/internal/types"

var VecDummy1 = types.EmbeddedVectorFloat32{
	0.125,
	0.25,
	0.5,
}

var VecDummy2 = types.EmbeddedVectorFloat32{
	0.375,
	0.75,
	0.875,
}

var VecDummy3 = types.EmbeddedVectorFloat32{
	0.875,
	0.9375,
	0.15625,
}

var VecDummy4 = types.EmbeddedVectorFloat32{
	0.15625,
	0.5,
	0.875,
}

const Base64Dummy1 = types.EmbeddedVectorBase64("AAAAPgAAgD4AAAA/")
const Base64Dummy2 = types.EmbeddedVectorBase64("AADAPgAAQD8AAGA/")
const Base64Dummy3 = types.EmbeddedVectorBase64("AABgPwAAcD8AACA+")
const Base64Dummy4 = types.EmbeddedVectorBase64("AAAgPgAAAD8AAGA/")
