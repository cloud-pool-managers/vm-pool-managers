# Image du JupyterLab collaboratif (temps réel / RTC-Yjs) pour les sessions de co-édition,
# hébergée sur la VM infra dédiée (colabVscodeInfra). Construite UNE fois sur cette VM :
#   sudo docker build -t collab-jupyter:latest -f collab-jupyter.Dockerfile .
# Versions figées : JupyterLab 4.2.7 + jupyter-collaboration 2.1.5 (couple compatible ; les
# versions plus récentes de l'image de base cassent l'extension de collaboration). On purge
# les fichiers d'extension périmés bakés dans l'image de base avant de réinstaller.
FROM registry.virtualdata.cloud.idcs.polytechnique.fr/docker-hub-proxy/jupyter/scipy-notebook:latest
USER root
RUN pip uninstall -y jupyter-collaboration jupyter-collaboration-ui jupyter-docprovider jupyter-server-ydoc 2>/dev/null || true; \
    rm -rf /opt/conda/share/jupyter/labextensions/@jupyter/collaboration-extension \
           /opt/conda/share/jupyter/labextensions/@jupyter/docprovider-extension
RUN pip install --no-cache-dir "jupyterlab==4.2.7" && \
    pip install --no-cache-dir --force-reinstall "jupyter-collaboration==2.1.5"
USER ${NB_UID}
