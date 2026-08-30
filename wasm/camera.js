// grmob-camera.js (WASM-specific Camera bridge)

window.GrMobCameraView = class {
    constructor(props) {
        this.props = props;
        this.videoElement = document.createElement("video");
        this.videoElement.autoplay = true;
        this.videoElement.style.width = "100%";
        this.videoElement.style.height = "100%";
        this.videoElement.style.objectFit = "cover";
        this.stream = null;

        this.init();
    }

    async init() {
        try {
            const facingMode = this.props.facing === "front" ? "user" : "environment";
            const constraints = {
                video: {
                    facingMode: { ideal: facingMode },
                }
            };

            this.stream = await navigator.mediaDevices.getUserMedia(constraints);
            this.videoElement.srcObject = this.stream;
        } catch (err) {
            console.error("Camera access error:", err);
            if (this.props.onError) {
                window.GoInvokeCallback(this.props.onError, err.message);
            }
        }
    }

    capture() {
        if (!this.videoElement || !this.stream) return;

        const canvas = document.createElement("canvas");
        canvas.width = this.videoElement.videoWidth;
        canvas.height = this.videoElement.videoHeight;
        const ctx = canvas.getContext("2d");
        ctx.drawImage(this.videoElement, 0, 0);
        const dataURL = canvas.toDataURL("image/png");

        if (this.props.onCapture) {
            window.GoInvokeCallback(this.props.onCapture, dataURL);
        }
    }

    stop() {
        if (this.stream) {
            this.stream.getTracks().forEach(track => track.stop());
        }
    }

    getElement() {
        return this.videoElement;
    }
};
