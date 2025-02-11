package testhelper

import "github.com/yammerjp/cachembed/internal/util"

var VecDummy1 = util.EmbeddedVectorFloat32{
	0.125,
	0.25,
	0.5,
}

var VecDummy2 = util.EmbeddedVectorFloat32{
	0.375,
	0.75,
	0.875,
}

var VecDummy3 = util.EmbeddedVectorFloat32{
	0.875,
	0.9375,
	0.15625,
}

var VecDummy4 = util.EmbeddedVectorFloat32{
	0.15625,
	0.5,
	0.875,
}

const Base64Dummy1 = util.EmbeddedVectorBase64("AAAAPgAAgD4AAAA/")
const Base64Dummy2 = util.EmbeddedVectorBase64("AADAPgAAQD8AAGA/")
const Base64Dummy3 = util.EmbeddedVectorBase64("AABgPwAAcD8AACA+")
const Base64Dummy4 = util.EmbeddedVectorBase64("AAAgPgAAAD8AAGA/")
