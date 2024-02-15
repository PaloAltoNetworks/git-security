export const showConfirmationDialog = (message) => {
    return ElMessageBox.confirm(message, 'Confirmation', {
        confirmButtonText: 'Yes',
        cancelButtonText: 'No',
        type: 'warning',
    })
}
