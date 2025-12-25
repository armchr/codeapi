package com.example.petclinic.controller;

import com.example.petclinic.dto.VisitDto;
import com.example.petclinic.service.VisitService;
import jakarta.validation.Valid;
import org.springframework.format.annotation.DateTimeFormat;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import java.time.LocalDate;
import java.util.List;

/**
 * REST controller for Visit operations.
 */
@RestController
@RequestMapping("/api/visits")
public class VisitController {

    private final VisitService visitService;

    public VisitController(VisitService visitService) {
        this.visitService = visitService;
    }

    /**
     * GET /api/visits - Get all visits.
     */
    @GetMapping
    public ResponseEntity<List<VisitDto>> getAllVisits() {
        return ResponseEntity.ok(visitService.findAll());
    }

    /**
     * GET /api/visits/{id} - Get visit by ID.
     */
    @GetMapping("/{id}")
    public ResponseEntity<VisitDto> getVisitById(@PathVariable Long id) {
        return ResponseEntity.ok(visitService.findById(id));
    }

    /**
     * GET /api/visits/pet/{petId} - Get visits for a pet.
     */
    @GetMapping("/pet/{petId}")
    public ResponseEntity<List<VisitDto>> getVisitsByPet(@PathVariable Long petId) {
        return ResponseEntity.ok(visitService.findByPetId(petId));
    }

    /**
     * GET /api/visits/vet/{vetId} - Get visits for a vet.
     */
    @GetMapping("/vet/{vetId}")
    public ResponseEntity<List<VisitDto>> getVisitsByVet(@PathVariable Long vetId) {
        return ResponseEntity.ok(visitService.findByVetId(vetId));
    }

    /**
     * GET /api/visits/today - Get today's visits.
     */
    @GetMapping("/today")
    public ResponseEntity<List<VisitDto>> getTodaysVisits() {
        return ResponseEntity.ok(visitService.findTodaysVisits());
    }

    /**
     * GET /api/visits/range - Get visits in date range.
     */
    @GetMapping("/range")
    public ResponseEntity<List<VisitDto>> getVisitsByDateRange(
            @RequestParam @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate startDate,
            @RequestParam @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate endDate) {
        return ResponseEntity.ok(visitService.findByDateRange(startDate, endDate));
    }

    /**
     * POST /api/visits - Schedule a new visit.
     */
    @PostMapping
    public ResponseEntity<VisitDto> scheduleVisit(@Valid @RequestBody VisitDto visitDto) {
        VisitDto created = visitService.scheduleVisit(visitDto);
        return new ResponseEntity<>(created, HttpStatus.CREATED);
    }

    /**
     * PATCH /api/visits/{id}/description - Update visit description.
     */
    @PatchMapping("/{id}/description")
    public ResponseEntity<VisitDto> updateDescription(@PathVariable Long id,
                                                       @RequestParam String description) {
        return ResponseEntity.ok(visitService.updateDescription(id, description));
    }

    /**
     * DELETE /api/visits/{id} - Cancel a visit.
     */
    @DeleteMapping("/{id}")
    public ResponseEntity<Void> cancelVisit(@PathVariable Long id) {
        visitService.cancelVisit(id);
        return ResponseEntity.noContent().build();
    }
}
