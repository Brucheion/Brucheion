/* On a quick note, this is kind of a shallow copy of the BrIC.js library that
 * is being shipped with Brucheion. However, BrIC is not suited for modular
 * use and is tied to the Go template setup of the first-version Brucheion. I've
 * decided to progress a bit and adopt parts of BrIC to flexible 2020 use.
 */

const localPath = '/static/image_archive/'
const localSuffix = '.dzi'

function getImagePathFromUrn(urn) {
  const ns = urn.split(':')[2]
  const collectionAndVersion = urn.split(':')[3]
  const collection = collectionAndVersion.split('.')[0]
  const version = collectionAndVersion.split('.')[1]

  return `${ns}/${collection}/${version}/`
}

function getTileSources(imgUrn) {
  const plainUrn = imgUrn.split('@')[0]
  const imgId = plainUrn.split(':')[4]
  const imagePath = getImagePathFromUrn(plainUrn)

  return localPath + imagePath + imgId + localSuffix
}

const defaultOpts = {
  crossOriginPolicy: 'Anonymous',
  minZoomImageRatio: 0.1, // of viewer size
  immediateRender: true,
  prefixUrl: '/static/css/images/',
}

export const getStaticOpts = (id, url) => ({
  ...defaultOpts,
  id,
  tileSources: {
    type: 'image',
    url,
  },
  buildPyramid: false,
  crossOriginPolicy: false,
})

export const getIIIFOpts = (id, imageManifest) => ({
  ...defaultOpts,
  id,
  tileSources: [imageManifest],
  sequenceMode: true,
  prefixUrl: '/static/css/images/',
  crossOriginPolicy: 'Anonymous',
  defaultZoomLevel: 1,
  minZoomImageRatio: 0.1, // of viewer size
  immediateRender: true,
})

export const getInternalOpts = (id, urn) => ({
  ...defaultOpts,
  id,
  tileSources: getTileSources(urn),
})
