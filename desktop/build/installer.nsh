; Фиксируем каталог установки, чтобы он не зависел от productName/name и не
; менялся от версии к версии. Раньше папка бралась из имени пакета
; ($LOCALAPPDATA\Programs\groovework-desktop); закрепляем её явно, иначе смена
; названия приложения увела бы установку в новый каталог, оставив старую копию
; и битые ярлыки. perMachine=false → пишем в HKCU (обе разрядности реестра).
!macro preInit
  SetRegView 64
  WriteRegExpandStr HKCU "${INSTALL_REGISTRY_KEY}" InstallLocation "$LOCALAPPDATA\Programs\groovework-desktop"
  SetRegView 32
  WriteRegExpandStr HKCU "${INSTALL_REGISTRY_KEY}" InstallLocation "$LOCALAPPDATA\Programs\groovework-desktop"
!macroend
