export const showConfirmationDialog = (message) => {
    return ElMessageBox.confirm(message, 'Confirmation', {
        confirmButtonText: 'Yes',
        cancelButtonText: 'No',
        type: 'warning',
    })
}

export const actionsConfirmationDialog = async (message) => {
    try {
    await ElMessageBox({
      message: message,
      showCancelButton: true,
      showConfirmButton: true,
      distinguishCancelAndClose: true,
      confirmButtonText: 'Enable',
      cancelButtonText: 'Disable',
      type: 'warning',
    });
    return true;
  } catch (action) {
    if (action == 'cancel') {
      return false; // Return false when Disable is clicked
    }
  }
}

export const showNotification = (status) => {
  if (status == "success") {
    ElNotification({
      title: "Success",
      message: "Operation success",
      type: "success",
      position: "bottom-right",
    });
  } else if (status == "error") {
    ElNotification({
      title: "Error",
      message: "Internal error occurred",
      type: "error",
      position: "bottom-right",
    });
  }
};