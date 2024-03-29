export const showConfirmationDialog = (message) => {
    return ElMessageBox.confirm(message, 'Confirmation', {
        confirmButtonText: 'Yes',
        cancelButtonText: 'No',
        type: 'warning',
    })
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