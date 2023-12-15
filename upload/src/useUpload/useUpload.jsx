import React, { useEffect, useRef } from 'react';
import PropTypes from 'prop-types';
import './useUpload.css';

const getSize = (value) => {
  if (!value) {
    return null;
  }

  const sizeRegex = /(\d+(?:\.\d+)?)\s*([a-zA-Z]+)/;
  const matches = value.toString().match(sizeRegex);

  if (!matches) {
    return value;
  }

  const [numericPart, unit] = matches;
  const numericValue = parseFloat(numericPart);

  switch (unit.toLowerCase()) {
    case 'gb':
    case 'g':
      return numericValue * 1024 * 1024 * 1024;
    case 'mb':
    case 'm':
      return numericValue * 1024 * 1024;
    case 'kb':
    case 'k':
      return numericValue * 1024;
    default:
      return null;
  }
};

export default function Upload({
  autoUpload,
  children,
  extensions,
  id,
  removeProgressbar,
  size,
  success,
  url,
  parameter,
}) {
  const instances = {};
  const uploadRef = useRef();
  const param = parameter || (Math.random() + 1).toString(36).substring(2);
  const message = (val) => {
    const value = val.replace(/\n/, '<br />');
    const messageElement = document.querySelector('.messagebox');
    const span = document.createElement('SPAN');

    span.innerHTML = `
        <span class="remove"></span>
        <span class="message">${value}</span>
         `;
    messageElement.appendChild(span);

    span.onclick = () => {
      span.parentNode.removeChild(span);
    };
  };

  const slugify = (filename) => {
    const ext = filename.match(/\.\w+$/gm);

    let fl = filename.normalize('NFD').replace(/[\u0300-\u036f]/g, '');
    fl = `${fl.replace(/\.\w+$/gm, '').replace(/[^\w\s-]/g, '').trim().replace(/\s+/g, '-')}_ft`;

    return fl.toLowerCase() + ext;
  };

  const removeFile = (idx, ins) => {
    ins[idx].element.remove();
    delete instances[idx];
  };

  const sliceUpload = (instance) => new Promise((resolve, reject) => {
    const data = new FormData();
    const filename = slugify(instance.file.name);
    const uploadId = instance.id;
    const { element } = instance;

    data.append('file', instance.file.slice(instance.start, instance.end, { type: 'application/octet-stream' }));
    data.append('type', 'multipart/form-data');
    fetch(instance.url, {
      method: 'POST',
      mode: 'cors',
      headers: {
        'X-id': uploadId,
        'X-Requested-With': 'XMLHttpRequest',
        'Cache-Control': 'no-cache',
        'X-File-Name': filename,
        'X-Slice': instance.current.toString(),
        'X-Slices': instance.slices.toString(),
        'X-File-Size': instance.file.size.toString(),
        'X-Slice-Size': instance.end.toString(),
        'X-Unique': instance.unique,
        'X-Parameter': param,
      },
      body: data,
    })
      .then((response) => {
        if (!response.ok) {
          reject(response.statusText);
        }
        return response.text();
      })
      .then((responseText) => {
        element.querySelector('.progress').style.width = `${instance.percentage}%`;
        element.querySelector('.filename > span').innerHTML = `${parseInt(instance.percentage, 10)}%`;
        let obj = {
          Error: undefined,
        };
        if (responseText !== '') {
          try {
            obj = JSON.parse(responseText);
          } catch (err) {
            message(err);
          }
        }
        if (obj.Error) {
          message(obj.Error);
        }
        resolve(obj);
      })
      .catch((error) => {
        reject(error);
      });
  });

  const fileUpload = async (i, ins, idx) => {
    const instance = ins[idx];
    let obj = {};
    if (i < instance.slices) {
      if (instance.running !== null && instance.running === true) {
        const k = i + 1;
        instance.current = k;
        instance.running = true;
        instance.start = i * instance.chunkSize;
        instance.end = (k === instance.slices) ? instance.file.size : (k * instance.chunkSize);
        instance.percentage = +((100 / instance.slices) * k);

        obj = await sliceUpload(instance);

        if (k === instance.slices) {
          if (instance.removeProgressbar === true) {
            removeFile(idx, ins);
          }
          instance.success(obj);
        } else {
          fileUpload(k, instances, idx).then(() => {});
        }
      }
    }
  };

  const handleFiles = (upload, event, files) => {
    const chunkSize = +(1024 * 1024);
    Array.from(files).forEach((file) => {
      const ext = file.name.split('.').pop().trim() || null;
      const idx = `file_${slugify(file.name)}_${file.size}_${performance.now()}`;
      const filesize = upload.size || 1099511627776;
      if (
        (upload.extensions.includes(ext) || upload.extensions.includes(ext.toLowerCase()))
        && files[id] === undefined
        && file.size <= filesize
      ) {
        const li = document.createElement('LI');

        li.innerHTML = `
          <div class="filename"><span>0%</span>${file.name}</div>
          <div class="progress"></div>
          <div class="control">
            <div class="pause"></div>
            <div class="start"></div>
            <div class="remove"></div>
          </div>`;
        upload.element.querySelector('ul').appendChild(li);

        instances[idx] = {
          id,
          file,
          element: li,
          slices: Math.ceil(file.size / chunkSize),
          current: null,
          running: true,
          chunkSize,
          url: upload.url,
          removeProgressbar: upload.removeProgressbar,
          success: upload.success,
          unique: `${file.size}${performance.now()}`,
          parameter: upload.parameter,
        };

        li.onclick = (e) => {
          const cl = Array.from(e.target.classList);
          if (cl.includes('remove')) {
            if (instances[idx] !== undefined) {
              instances[idx].running = null;
            }
            removeFile(idx, instances);
          }
        };
        if (upload.autoUpload === true) {
          fileUpload(0, instances, idx).then(() => {
          });
        } else {
          const btn = upload.element.querySelector('.uploadBtn');
          btn.style.display = 'block';
          btn.onclick = (e) => {
            Object.keys(instances).forEach((key) => {
              fileUpload(0, instances, key).then(() => {
              });
              e.preventDefault(instances[key]);
            });
            btn.style.display = 'none';
          };
        }
      } else {
        message('Wrong file format or oversize file!');
      }
    });
  };

  const createNewInstance = (upload) => {
    [].map.call(upload.element.querySelectorAll('input[type=file],.dropArea'), (el) => {
      if (!el.dataset.bind) {
        el.setAttribute('data-bind', true);
        el.addEventListener('change', (e) => {
          handleFiles(upload, e, el.files);
          e.preventDefault();
        });

        if (el.tagName === 'DIV' && el.classList.contains('dropArea')) {
          const dropArea = el;

          const tests = {
            fileReader: typeof FileReader !== 'undefined',
            draggable: 'draggable' in document.createElement('span'),
            formData: !!window.FormData,
          };

          dropArea.onclick = (element) => {
            element.target.closest('.fileUpload').querySelector('input[type=file]').click();
          };

          if (tests.draggable) {
            dropArea.ondragover = () => {
              dropArea.classList.add('hover');
              return false;
            };

            dropArea.ondragend = () => {
              dropArea.classList.remove('hover');
              return false;
            };

            dropArea.ondragleave = () => {
              dropArea.classList.remove('hover');
              return false;
            };

            dropArea.ondrop = (e) => {
              dropArea.classList.remove('hover');
              e.preventDefault();
              const files = e.target.files || e.dataTransfer.files;
              handleFiles(upload, e, files);
              return false;
            };
          }
        }
      }
    });
  };

  useEffect(() => {
    if (!window.upload || !(window.File && window.FileReader && window.FileList && window.Blob)) {
      window.upload = {};
    }

    const upload = {
      id,
      url,
      success,
      autoUpload,
      removeProgressbar,
      files: {},
      size: getSize(size),
      element: uploadRef.current,
      extensions: (extensions || null) !== null ? extensions.replace(/jpg/gi, 'jpg jpeg').trim().split(' ') : null,
    };

    createNewInstance(upload);
  }, []);

  return (
    <section ref={uploadRef} className="fileUpload">
      <div className="messagebox" />
      <div className="data">
        <span className="info">{children}</span>
        <input type="file" multiple accept="" />
      </div>
      <div className="dropArea" />
      <div className="filePreview" />
      <ul />
      <hr />
      <div className="uploadBtn">Upload</div>
    </section>
  );
}

Upload.defaultProps = {
  id: null,
  removeProgressbar: true,
  autoUpload: true,
  size: 1024 * 1024 * 32,
  extensions: 'jpg gif webp png',
  children: 'File upload',
  parameter: '',
  success: PropTypes.func,
};

Upload.propTypes = {
  id: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
  url: PropTypes.string.isRequired,
  removeProgressbar: PropTypes.bool,
  autoUpload: PropTypes.bool,
  size: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
  extensions: PropTypes.string,
  success: PropTypes.func,
  children: PropTypes.node,
  parameter: PropTypes.string,
};
