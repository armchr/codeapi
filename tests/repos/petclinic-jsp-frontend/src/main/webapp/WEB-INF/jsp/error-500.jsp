<%@ page contentType="text/html;charset=UTF-8" isErrorPage="true" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="includes/header.jsp">
    <jsp:param name="title" value="Server Error"/>
</jsp:include>

<div class="row justify-content-center">
    <div class="col-md-6 text-center">
        <div class="display-1 text-danger mb-4">500</div>
        <h2>Internal Server Error</h2>
        <p class="text-muted">Something went wrong on our end. Please try again later.</p>
        <a href="${pageContext.request.contextPath}/home" class="btn btn-primary">
            <i class="bi bi-house me-1"></i>Go Home
        </a>
    </div>
</div>

<jsp:include page="includes/footer.jsp"/>
