import React from 'react'
import Upload from '../src/useUpload/useUpload.jsx'

function App () {
  const SuccessFunction = (response) => {
    console.log(response)
  }

  return (
    <>
      <Upload id={null} url={'/api/upload'} removeProgressbar={true} autoUpload={true} size={1024 * 1024 * 32} extensions={'jpg gif webp png'} success={SuccessFunction}>Image upload: Max 32mb</Upload>
    </>
  )
}

export default App
