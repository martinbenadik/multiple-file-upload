# React + Vite

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react/README.md) uses [Babel](https://babeljs.io/) for Fast Refresh
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react-swc) uses [SWC](https://swc.rs/) for Fast Refresh

# Multi-File Upload Component Documentation

## Overview

The Multi-File Upload component is a React component designed for handling multiple file uploads with features such as automatic upload, file type restrictions, and progress tracking.

## Table of Contents

- [Usage](#usage)
- [Props](#props)
- [Styling](#styling)
- [Contributing](#contributing)
- [License](#license)

# Usage

```javascript
import React from 'react';
import Upload from '../src/useUpload/useUpload.jsx'

function App() {
  const successFunction = (response) => {
    console.log(response);
  }

  return (
    <>
      <Upload
        id={null}
        url={'/api/upload'}
        removeProgressbar={false}
        autoUpload={true}
        size={1024 * 1024 * 32}
        extensions={'jpg gif webp png'}
        success={successFunction}
      >
        Image upload: Max 32mb
      </Upload>
    </>
  );
}

export default App;
```

# Props

- id (optional): Unique identifier for the file upload component.
- url (required): The server endpoint for file uploads.
- removeProgressbar (optional): Determines whether to remove the progress bar after upload completion.
- autoUpload (optional): Enables automatic file uploads upon file selection.
- size (optional): Maximum allowed file size in bytes.
- extensions (optional): Allowed file extensions (e.g., 'jpg gif webp png').
- success (required): Callback function invoked upon successful file upload.

# Styling

The Multi-File Upload component provides basic styling, but you can customize the appearance by modifying the provided CSS styles. Ensure to import the styles into your project.

# Contributing

We welcome contributions! If you find a bug or have an enhancement in mind, please open an issue or submit a pull request.

# License

This project is licensed under the MIT License.

MIT License



